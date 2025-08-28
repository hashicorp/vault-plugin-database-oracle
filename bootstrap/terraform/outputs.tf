# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

output "info" {
  value = <<EOF

Vault Server IP (public): ${aws_instance.vault-server.public_ip}
Vault UI URL:             http://${aws_instance.vault-server.public_ip}:8200/ui

You can SSH into the Vault EC2 instance using private.key:
    ssh -i private.key ubuntu@${aws_instance.vault-server.public_ip}

EOF
}
