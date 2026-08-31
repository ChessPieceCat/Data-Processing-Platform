terraform {
  required_version = ">= 1.16.0"

  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
  }

  backend "s3" {
    bucket       = "data-processing-platform-terraform-state-410126553529"
    key          = "data-processing-platform/terraform.tfstate"
    region       = "us-east-2"
    use_lockfile = true
  }
}

provider "aws" {
  region = var.aws_region
}

data "aws_vpc" "default" {
  default = true
}

data "aws_subnet" "app" {
  id = var.app_subnet_id
}