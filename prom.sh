#!/usr/bin/env bash


bob trigger-promotion \
  --product-name vault-plugin-database-oracle \
  --org hashicorp --repo vault-plugin-database-oracle --branch main \
  --product-version "0.8.3" \
  --sha "$(git rev-parse HEAD)" \
  --environment vault-plugin-database-oracle-oss \
  --slack-channel C03RXFX5M4L \
  production

