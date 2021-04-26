# tf-measure-refresh

Measure the refresh time for a specific resource type in a Terraform workspace.

## How it works

Make a temporary directory. Use `jq` to parse the terraform.tfstate file, pull out
only the resources we want to measure and copy them to the temporary directory. Run
`terraform refresh` in the temporary directory and measure the time it takes.


