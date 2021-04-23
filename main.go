package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

var providerConfig = `
provider "aviatrix" {
  skip_version_validation = true
}
terraform {
  required_providers {
    aviatrix = {
      source = "aviatrixsystems/aviatrix"
    }
  }
}
`

func main() {
	if len(os.Args) != 2 {
		log.Fatal("measureRefresh takes a single argument")
	}
	if _, err := os.Stat("terraform.tfstate"); err != nil {
		log.Fatalf("could not find terraform.tfstate file in current dir: %v", err)
	}

	log.Printf("=> Measuring Refresh Time for all %s resources\n", os.Args[1])
	log.Println("=> Making Temp Dir")
	dir, err := os.MkdirTemp("", "measureRefresh")
	if err != nil {
		log.Fatalf("making temp dir: %v", err)
	}
	defer os.RemoveAll(dir) // clean up
	log.Println("=> Made Temp Dir:", dir)

	log.Println("=> Using jq to get resources from tfstate")
	jqArgs := []string{
		fmt.Sprintf("del(.resources[] | select(.type != %q))", os.Args[1]),
		"terraform.tfstate",
	}
	jq := exec.Command("jq", jqArgs...)
	out, err := jq.Output()
	if err != nil {
		exitErr := err.(*exec.ExitError)
		log.Fatalf("parsing tfstate: %v jq stderr: %s", err, exitErr.Stderr)

	}

	log.Println("=> Writing new statefile to temp dir")
	err = os.WriteFile(filepath.Join(dir, "terraform.tfstate"), out, 0666)
	if err != nil {
		log.Fatalf("writing new statefile: %v", err)
	}

	log.Println("=> Copying over tf files to temp dir")
	cp := exec.Command("/bin/sh", "-c", "cp *.tf "+dir)
	out, err = cp.CombinedOutput()
	if err != nil {
		log.Fatalf("could not cp tf files to temp dir: %v\n cp out: %s", err, out)
	}

	log.Println("=> Running terraform init")
	c := exec.Command("terraform", "init")
	c.Dir = dir
	err = c.Run()
	if err != nil {
		log.Fatalf("running terraform init: %v", err)
	}

	log.Println("=> Removing *.tf files")
	rm := exec.Command("/bin/sh", "-c", "rm *.tf")
	rm.Dir = dir
	out, err = rm.CombinedOutput()
	if err != nil {
		log.Fatalf("could not rm tf files in temp dir: %v\n rm out: %s", err, out)
	}

	log.Println("=> Writing temp config file")
	err = os.WriteFile(filepath.Join(dir, "main.tf"), []byte(providerConfig), 0666)
	if err != nil {
		log.Fatalf("writing temp config: %v", err)
	}

	log.Println("=> Running terraform refresh and measuring execution time")
	c = exec.Command("terraform", "refresh")
	c.Dir = dir
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	start := time.Now()
	err = c.Run()
	end := time.Now()
	if err != nil {
		log.Fatalf("running terraform refresh: %v", err)
	}

	total := end.Sub(start)
	log.Println("=> Total time to refresh:", total)
}
