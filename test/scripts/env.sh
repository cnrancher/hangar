#!/bin/bash

set -euo pipefail

# Configuration for Registry
export HANGAR_REGISTRY_NAME="hangar-registry"
export HANGAR_REGISTRY_PORT=5000

# Set this environment variable to avoid the permission denined of
# mkdir /run/containers
export REGISTRY_AUTH_FILE="${HOME}/.config/containers/auth.json"
