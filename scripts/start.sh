#!/usr/bin/env bash
set -e

ENV=${1:-local}
VALID_ENVS="local eng prod"

if [[ ! " $VALID_ENVS " =~ " $ENV " ]]; then
  echo "Invalid environment: $ENV"
  echo "Usage: $0 [local|eng|prod]"
  exit 1
fi

ENV_FILE="$(dirname "$0")/../env/${ENV}.env"
if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing env file: $ENV_FILE"
  echo "Copy env/${ENV}.env.example to env/${ENV}.env and edit as needed."
  exit 1
fi

set -a
source "$ENV_FILE"
set +a

BIN_DIR="$(dirname "$0")/../bin"
SERVER="$BIN_DIR/server"
if [[ ! -f "$SERVER" ]]; then
  echo "Binary not found: $SERVER"
  echo "Run 'make build' first."
  exit 1
fi

exec "$SERVER"
