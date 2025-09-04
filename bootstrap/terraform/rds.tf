# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

locals {
  db_connection_string = "${aws_db_instance.main.endpoint}/${aws_db_instance.main.db_name}"
}

# password for the master DB user
resource "random_password" "db_admin" {
  length  = 30
  special = false
}

# password for the root Vault user
resource "random_password" "vault_admin" {
  length  = 30
  special = false
}

resource "aws_db_instance" "main" {
  identifier                 = "${var.environment_name}-db"
  allocated_storage          = 20
  auto_minor_version_upgrade = true
  db_name                    = var.oracle_db_name
  engine                     = "oracle-se2"
  engine_version             = "19.0.0.0.ru-2025-07.spb-1.r1"
  instance_class             = var.db_instance_type
  license_model              = "license-included"
  password                   = random_password.db_admin.result
  publicly_accessible        = true
  skip_final_snapshot        = true
  storage_type               = "gp2"
  username                   = var.db_admin
  vpc_security_group_ids     = [aws_security_group.rds.id]
}

