package main

import (
	"flag"
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

var runs int

func init() {
	flag.IntVar(&runs, "runs", 1, "how many terraform refresh runs to measure to get the average time")
}

func main() {
	flag.Parse()
	if len(flag.Args()) != 1 {
		log.Fatal("measureRefresh takes a single argument")
	}
	if _, err := os.Stat("terraform.tfstate"); err != nil {
		log.Fatalf("could not find terraform.tfstate file in current dir: %v", err)
	}
	if matches, _ := filepath.Glob("*.tf"); len(matches) == 0 {
		log.Fatal("no *.tf files found in the current directory")
	}
	if _, err := exec.LookPath("jq"); err != nil {
		log.Fatal("jq must be in your PATH")
	}

	log.Printf("=> Measuring Refresh Time for all %s resources\n", flag.Arg(0))
	log.Println("=> Making Temp Dir")
	dir, err := os.MkdirTemp("", "measureRefresh")
	if err != nil {
		log.Fatalf("making temp dir: %v", err)
	}
	defer os.RemoveAll(dir) // clean up
	log.Println("=> Made Temp Dir:", dir)

	log.Println("=> Using jq to get resources from tfstate")
	jqArgs := []string{
		fmt.Sprintf("del(.resources[] | select(.type != %q))", flag.Arg(0)),
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

	log.Printf("=> Running terraform refresh %d times and measuring average execution time", runs)
	var sum int64
	for i := 0; i < runs; i++ {
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
		sum += total.Milliseconds()
	}
	avg := sum / int64(runs)
	log.Println("=> Total time to refresh:", float64(avg)/1000.0)
}
