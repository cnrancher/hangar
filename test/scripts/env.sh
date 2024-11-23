#!/bin/bash

set -euo pipefail

# Configuration for Harbor
export HARBOR_HELM_VERSION="v1.16.0"
export HARBOR_URL=${HARBOR_URL:-'localhost'}
export HARBOR_PORT="443" # ingress https
export HARBOR_PASSWORD="testpassword123"
export K3S_CLUSTER_NAME="testharbor"

# Configuration for Distribution Registry
export DISTRIBUTION_URL="127.0.0.1:5000"

# Set this environment variable to avoid the permission denined of
# mkdir /run/containers
export REGISTRY_AUTH_FILE="${HOME}/.config/containers/auth.json"
