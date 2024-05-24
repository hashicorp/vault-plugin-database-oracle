#!/usr/bin/env bash

# usage:
#     ./release.sh [VERSION] [BRANCH] [ENVIRONMENT]

if [[ "$#" != 3 ]]; then
  echo "
usage:
     ./release.sh [VERSION] [BRANCH] [ENVIRONMENT]

example usage:
     ./release.sh 0.7.0 main [staging|production]
  "
  exit 1
fi

if [[ "$3" != +(staging|production) ]]; then
  echo "[error] environment must be one of 'staging' or 'production'"
  exit 1
fi

VERSION="$1"
BRANCH="$2"
ENVIRONMENT="$3"

git checkout $BRANCH
git pull

bob trigger-promotion \
  --product-name vault-plugin-database-oracle \
  --org hashicorp \
  --repo vault-plugin-database-oracle \
  --branch $BRANCH \
  --product-version $VERSION \
  --sha "$(git rev-parse HEAD)" \
  --environment vault-plugin-database-oracle-oss \
  --slack-channel C03RXFX5M4L \
  $ENVIRONMENT

