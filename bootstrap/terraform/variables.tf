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

# db_admin is the username for the master DB user. DO NOT provide this to Vault.
variable "db_admin" {
  default = "DB_ADMIN"
}

# vault_admin is the username for the Vault root user.
variable "vault_admin" {
  default = "VAULT_ADMIN"
}

# num_static_users is the number of static DB users we will create for our test environment
variable "num_static_users" {
  default = 10
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

