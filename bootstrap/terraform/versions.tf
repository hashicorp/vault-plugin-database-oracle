# Copyright IBM Corp. 2017, 2025
# SPDX-License-Identifier: MPL-2.0


terraform {
  required_version = ">= 1.5"
  required_providers {
    aws = {
      version = "~> 5.36.0"
    }
  }
}
