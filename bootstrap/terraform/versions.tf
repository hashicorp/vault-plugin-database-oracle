# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


terraform {
  required_version = ">= 1.5"
  required_providers {
    aws = {
      version = "~> 5.36.0"
    }
  }
}
