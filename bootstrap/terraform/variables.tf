# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

# AWS region and AZs in which to deploy
variable "aws_region" {
  default = "us-east-1"
}

# All resources will be tagged with this
variable "environment_name" {
  default = "vault-plugin-database-oracle-demo"
}

# URL for Vault ENT binary
variable "vault_zip_file" {
  default = "https://releases.hashicorp.com/vault/1.19.1+ent/vault_1.19.1+ent_linux_amd64.zip"
}

variable "vault_license" {
}

variable "vault_admin" {
  default = "vaultadmin"
}

# Instance size
variable "instance_type" {
  default = "t3.large"
}

# DB instance size
variable "db_instance_type" {
  default = "db.t3.medium"
}

variable "oracle_db_name" {
  default = "ORACLEDB"
}

# Oracle plugin version
variable "oracle_plugin_version" {
  default = "0.10.2"
}

