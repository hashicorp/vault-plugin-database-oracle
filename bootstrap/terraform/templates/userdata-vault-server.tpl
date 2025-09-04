#!/usr/bin/env bash

# NOTE: This is a Terraform template file. See a note on
# escaping the '$' character in the ./README.md.

set -ex

exec > >(tee /var/log/tf-user-data.log|logger -t user-data ) 2>&1

##--------------------------------------------------------------------
## Functions

logger() {
  set +x 2>/dev/null
  DT=$(date '+%Y/%m/%d %H:%M:%S')
  echo "[DEBUG] $DT: $1"
  set -x
}

fail() {
  logger "$1" 1>&2
  exit 1
}

user_ubuntu() {
  # UBUNTU user setup
  if ! getent group $${USER_GROUP} >/dev/null
  then
    sudo addgroup --system $${USER_GROUP} >/dev/null
  fi

  if ! getent passwd $${USER_NAME} >/dev/null
  then
    sudo adduser \
      --system \
      --disabled-login \
      --ingroup $${USER_GROUP} \
      --home $${USER_HOME} \
      --no-create-home \
      --gecos "$${USER_COMMENT}" \
      --shell /bin/false \
      $${USER_NAME}  >/dev/null
  fi

}

logger "Running"

##--------------------------------------------------------------------
## Variables

# Get Private IP address
# 1. Fetch a session token and store it in a variable.
TOKEN=$(curl -X PUT "http://169.254.169.254/latest/api/token" -H "X-aws-ec2-metadata-token-ttl-seconds: 21600")
# 2. Use the token in a header to get the private IP address.
PRIVATE_IP=$(curl -H "X-aws-ec2-metadata-token: $TOKEN" http://169.254.169.254/latest/meta-data/local-ipv4)

VAULT_ZIP="${tpl_vault_zip_file}"

AWS_REGION="${tpl_aws_region}"
KMS_KEY="${tpl_kms_key}"

BIN="/usr/local/bin"
VAULT_BIN="$${BIN}/vault"
USER_NAME="vault"
USER_COMMENT="HashiCorp Vault user"
USER_GROUP="vault"
USER_HOME="/srv/vault"

SYSTEMD_DIR="/lib/systemd/system"

# Variables for the plugin
PLUGIN_NAME="vault-plugin-database-oracle"
PLUGIN_VERSION="${tpl_oracle_plugin_version}"
VAULT_DIR="/etc/vault.d"
PLUGIN_DIR="$${VAULT_DIR}/plugins"
ORACLE_PLUGIN_PATH="$${PLUGIN_DIR}/$${PLUGIN_NAME}"
ORACLE_DIR="/opt/oracle"

##--------------------------------------------------------------------
## Install Base Prerequisites

logger "Setting timezone to UTC"
sudo timedatectl set-timezone UTC

logger "Performing updates and installing prerequisites"
sudo apt-get -qq -y update
# libaio1 is required by the Oracle Instant Client libs
sudo apt-get install -qq -y wget unzip jq libaio1

logger "Disable reverse dns lookup in SSH"
sudo sh -c 'echo "\nUseDNS no" >> /etc/ssh/sshd_config'
sudo systemctl restart ssh

##--------------------------------------------------------------------
## Install Vault

logger "Downloading Vault"
curl -o /tmp/vault.zip "$${VAULT_ZIP}"

logger "Installing Vault"
sudo unzip -o /tmp/vault.zip -d "$${BIN}/"
sudo mkdir -pm 0755 "$${VAULT_DIR}"
sudo mkdir -pm 0755 /etc/ssl/vault

# Create the plugin directory
sudo mkdir -pm 0755 "$${PLUGIN_DIR}"

logger "$${VAULT_BIN} --version: $($${VAULT_BIN} --version)"

logger "Configuring Vault"
sudo tee "$${VAULT_DIR}/vault.hcl" <<EOF
storage "raft" {
  path    = "$${USER_HOME}/data"
  node_id = "vault-node-1"
}

listener "tcp" {
  address = "$${PRIVATE_IP}:8200"
  tls_disable = 1
}

api_addr = "http://$${PRIVATE_IP}:8200"
cluster_addr = "http://$${PRIVATE_IP}:8201"

seal "awskms" {
  region = "$${AWS_REGION}"
  kms_key_id = "$${KMS_KEY}"
}

license_path = "$${VAULT_DIR}/vault.hclic"
plugin_directory = "$${PLUGIN_DIR}"

ui=true
EOF

sudo tee -a /etc/environment <<EOF
export VAULT_ADDR=http://$${PRIVATE_IP}:8200
export VAULT_SKIP_VERIFY=true
EOF

source /etc/environment

logger "Creating vault license file"
sudo tee -a "$${VAULT_DIR}/vault.hclic" <<EOF
${tpl_vault_license}
EOF

##--------------------------------------------------------------------
## Install Oracle Instant Client
logger "Installing Oracle Instant Client"

# Create a standard location for Oracle software
sudo mkdir -p "$${ORACLE_DIR}"

wget https://download.oracle.com/otn_software/linux/instantclient/1928000/instantclient-basic-linux.x64-19.28.0.0.0dbru.zip \
  -P "$${ORACLE_DIR}"

wget https://download.oracle.com/otn_software/linux/instantclient/1928000/instantclient-sqlplus-linux.x64-19.28.0.0.0dbru.zip \
  -P "$${ORACLE_DIR}"

# This will create a versioned directory, such as ORACLE_DIR/instantclient_19_28
cd "$${ORACLE_DIR}"
sudo unzip -o '*.zip'

# Determine the directory name dynamically
ORACLE_CLIENT_DIR=$(sudo find /opt/oracle -maxdepth 1 -type d -name "instantclient_*")
if [[ -z "$${ORACLE_CLIENT_DIR}" ]]; then
  fail "Error: Oracle Instant Client directory not found."
fi

# Create a config file pointing to the Oracle library directory
echo "$${ORACLE_CLIENT_DIR}" | sudo tee /etc/ld.so.conf.d/oracle-instantclient.conf

# Update the linker cache
sudo ldconfig

if [[ -f "$${ORACLE_CLIENT_DIR}/sqlplus" ]]; then
  # Create a symbolic link in /usr/local/bin to the sqlplus executable
  sudo ln -sf "$${ORACLE_CLIENT_DIR}/sqlplus" /usr/local/bin/sqlplus
  logger "Successfully created symbolic link for sqlplus."
else
  fail "Error: sqlplus executable not found in $${ORACLE_CLIENT_DIR}."
fi

logger "Creating Vault environment file $${VAULT_DIR}/vault.env"
sudo tee "$${VAULT_DIR}/vault.env" > /dev/null <<EOF
# empty
EOF

cat "$${VAULT_DIR}/vault.env"

##--------------------------------------------------------------------
## Configure Vault user

logger "Setting up user $${USER_NAME} for Debian/Ubuntu"
user_ubuntu

logger "Granting ubuntu user sudo access to the vault user"
echo "ubuntu ALL=(vault) NOPASSWD: ALL" | sudo tee /etc/sudoers.d/99-vault-user
sudo chmod 0440 /etc/sudoers.d/99-vault-user

##--------------------------------------------------------------------
## Install Vault Systemd Service

sudo tee "$${SYSTEMD_DIR}/vault.service" > /dev/null <<EOF
[Unit]
Description="HashiCorp Vault - A tool for managing secrets"
Documentation=https://www.vaultproject.io/docs/
Requires=network-online.target
After=network-online.target
ConditionFileNotEmpty=$${VAULT_DIR}/vault.hcl
StartLimitIntervalSec=60
StartLimitBurst=3

[Service]
Type=notify
User=vault
Group=vault
ProtectSystem=full
ProtectHome=read-only
PrivateTmp=yes
PrivateDevices=yes
SecureBits=keep-caps
AmbientCapabilities=CAP_IPC_LOCK
CapabilityBoundingSet=CAP_SYSLOG CAP_IPC_LOCK
NoNewPrivileges=yes
ExecStart=$${VAULT_BIN} server -config=$${VAULT_DIR}/vault.hcl -log-level=debug
ExecReload=/bin/kill --signal HUP \$MAINPID
KillMode=process
KillSignal=SIGINT
Restart=on-failure
RestartSec=5
TimeoutStopSec=30
LimitNOFILE=65536
LimitMEMLOCK=infinity
LimitCORE=0
EnvironmentFile=$${VAULT_DIR}/vault.env

[Install]
WantedBy=multi-user.target
EOF

logger "Installing systemd services for Debian/Ubuntu"

# Create the vault directories
sudo mkdir -p "$${USER_HOME}/data"

# Set ownership for ALL Vault-related paths
sudo chown -R vault:vault "$${VAULT_DIR}" /etc/ssl/vault "$${USER_HOME}" "$${ORACLE_CLIENT_DIR}"

# Set secure permissions
# Dirs: rwxr-x--- (750), Files: rw-r----- (640)
sudo find "$${VAULT_DIR}" -type d -exec chmod 750 {} +
sudo find "$${VAULT_DIR}" -type f -exec chmod 640 {} +

# Set final permissions for the service file and binary
sudo chmod 0644 "$${SYSTEMD_DIR}/vault.service"
sudo chmod 0755 "$${VAULT_BIN}"

sudo systemctl enable vault
sudo systemctl start vault

# disable exit on error for vault status check
set +e

logger "Waiting for vault to be ready"
vault status
while [[ $? -ne 2 ]]; do sleep 1 && vault status; done

# enable error on exit
set -e

##--------------------------------------------------------------------
## Configure Vault
##--------------------------------------------------------------------

# NOT SUITABLE FOR PRODUCTION USE
export VAULT_TOKEN="$(vault operator init -format json | jq -r '.root_token')"
sudo cat >> /etc/environment <<EOF
export VAULT_TOKEN="$${VAULT_TOKEN}"
EOF

##--------------------------------------------------------------------
## Install Vault Oracle Database Plugin

logger "Installing Vault Oracle Database Plugin"

logger "Downloading Oracle plugin version $${PLUGIN_VERSION}"
curl -s -o "/tmp/$${PLUGIN_NAME}.zip" \
  "https://releases.hashicorp.com/$${PLUGIN_NAME}/$${PLUGIN_VERSION}/$${PLUGIN_NAME}_$${PLUGIN_VERSION}_linux_amd64.zip"

sudo unzip -o "/tmp/$${PLUGIN_NAME}.zip" -d $${PLUGIN_DIR}
sudo chown -R vault:vault "$${PLUGIN_DIR}"
sudo chmod 0755 "$${ORACLE_PLUGIN_PATH}"

logger "Verifying dynamic libs"
sudo -u vault ldd /etc/vault.d/plugins/vault-plugin-database-oracle

PLUGIN_SHA256=$(sha256sum "$${ORACLE_PLUGIN_PATH}" | cut -d ' ' -f 1)

vault plugin register -sha256="$${PLUGIN_SHA256}" database "$${PLUGIN_NAME}"

logger "Complete"

# There is a remote-exec provisioner in terraform watching for this file
touch /tmp/user-data-completed
