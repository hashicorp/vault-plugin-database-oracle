# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

//--------------------------------------------------------------------
// Vault Server Instance

resource "aws_instance" "vault-server" {
  ami                         = data.aws_ssm_parameter.ubuntu_ami.value
  instance_type               = var.instance_type
  key_name                    = aws_key_pair.main.key_name
  vpc_security_group_ids      = [aws_security_group.vault-server.id]
  associate_public_ip_address = true
  iam_instance_profile        = aws_iam_instance_profile.vault-server.id

  tags = {
    Name = "${var.environment_name}-vault-server"
  }

  user_data = templatefile(
    "${path.module}/templates/userdata-vault-server.tpl",
    {
      tpl_vault_zip_file        = var.vault_zip_file
      tpl_oracle_plugin_version = var.oracle_plugin_version
      tpl_kms_key               = aws_kms_key.vault.id
      tpl_aws_region            = var.aws_region
      tpl_vault_license         = var.vault_license
  })

  # Bit of a hack to wait for user_data script to finish running before returning
  provisioner "remote-exec" {
    inline = [
      "while [ ! -f /tmp/user-data-completed ]; do sleep 2; done",
    ]

    connection {
      type        = "ssh"
      user        = "ubuntu"
      host        = aws_instance.vault-server.public_ip
      private_key = tls_private_key.main.private_key_pem
    }
  }

  lifecycle {
    ignore_changes = [
      ami,
      tags,
    ]
  }
}

data "aws_caller_identity" "current" {
}

resource "null_resource" "db_create_static_users" {
  depends_on = [aws_instance.vault-server, aws_db_instance.main]

  triggers = {
    # If the content of a trigger changes, its MD5 hash will change,
    # forcing this resource to be replaced.
    create_sh_hash  = filemd5("${path.module}/scripts/create.sh")
    create_sql_hash = filemd5("${path.module}/scripts/create.sql")
  }

  connection {
    type        = "ssh"
    user        = "ubuntu" # Default user for Ubuntu AMIs
    host        = aws_instance.vault-server.public_ip
    private_key = tls_private_key.main.private_key_pem
  }

  provisioner "remote-exec" {
    inline = [
      "rm -rf /home/ubuntu/scripts",
      "mkdir -p /home/ubuntu/scripts"
    ]
  }

  # Upload scripts directory to the server's home folder.
  provisioner "file" {
    source      = "scripts/"
    destination = "/home/ubuntu/scripts"
  }

  provisioner "remote-exec" {
    inline = [<<-EOT
      #!/bin/bash
      set -ex

      chmod +x /home/ubuntu/scripts/create.sh

      # Export environment variables for the script to use.
      export CONN_URL="${var.db_admin}/${random_password.db_admin.result}@//${local.db_connection_string}"
      export VAULT_ADMIN="${var.vault_admin}"
      export QUERY_FILE_PATH="/home/ubuntu/scripts/create.sql"
      export VAULT_ADMIN_PASSWORD="${random_password.vault_admin.result}"
      export NUM_STATIC_USERS="${var.num_static_users}"

      # Now, execute the script.
      /home/ubuntu/scripts/create.sh
    EOT
    ]
  }
}

# Ideally, we would use TFVP for this, but then we will need to expose the
# Vault server to the public internet.
resource "null_resource" "vault-configure-plugin" {
  depends_on = [aws_instance.vault-server, aws_db_instance.main, null_resource.db_create_static_users]

  connection {
    type        = "ssh"
    user        = "ubuntu" # Default user for Ubuntu AMIs
    host        = aws_instance.vault-server.public_ip
    private_key = tls_private_key.main.private_key_pem
  }

  provisioner "remote-exec" {
    inline = [<<-EOT
      #!/bin/bash
      set -e

      # Safety check, wait for vault to be ready
      while ! vault status > /dev/null 2>&1; do
        sleep 2
      done

      # Enable only if it's not already enabled to make the script idempotent
      if ! vault secrets list | grep -q "database/"; then
        vault secrets enable database
      fi

      vault write database/config/oracle \
        plugin_name="vault-plugin-database-oracle" \
        allowed_roles="*" \
        connection_url="{{username}}/{{password}}@//${local.db_connection_string}" \
        username="${var.vault_admin}" \
        password="${random_password.vault_admin.result}" \
        verify_connection=true

      vault write database/roles/test \
        db_name="oracle" \
        default_ttl="1h" max_ttl="24h" \
        creation_statements="CREATE USER {{name}} IDENTIFIED BY \"{{password}}\"; GRANT CONNECT TO {{name}};"

      for i in $(seq 0 ${var.num_static_users}); do
        vault write "database/static-roles/static-role-$${i}" \
          db_name=oracle \
          username="STATIC_USER_$${i}" \
          rotation_statements="ALTER USER \"{{username}}\" IDENTIFIED BY \"{{password}}\";" \
          rotation_period=10m
      done
    EOT
    ]
  }
}
