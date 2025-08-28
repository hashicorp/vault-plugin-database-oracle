# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

//--------------------------------------------------------------------
// Providers

provider "aws" {
  // Credentials set via env vars
  region = var.aws_region
}

//--------------------------------------------------------------------
// Data Sources

data "aws_ssm_parameter" "ubuntu_ami" {
  name = "/aws/service/canonical/ubuntu/server/22.04/stable/current/amd64/hvm/ebs-gp2/ami-id"
}
