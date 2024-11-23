#!/bin/bash

set -euo pipefail

cd $(dirname $0)/../
WORKINGDIR="$(pwd)"

source scripts/env.sh
source scripts/harbor.sh
source scripts/distribution.sh

function usage() {
    echo "Run validation test for Hangar."
    echo "Usage:"
    echo "    $0 [OPTION]"
    echo "Option:"
    echo "    --harbor                   Run tests for Harbor registry server"
    echo "    --distribution-registry    Run tests for Distribution registry"
    echo "    --all                      Run tests for Harbor & Distribution registry"
    echo "    --help       --            Show this message"
}

while [[ $# -gt 0 ]]; do
  case $1 in
    --harbor)
        HARBOR=1
        shift # past argument
        ;;
    --distribution-registry)
        DISTRIBUTION=1
        shift # past argument
        ;;
    --all)
        HARBOR=1
        DISTRIBUTION=1
        shift # past argument
        ;;
    -h|--help)
        usage
        exit 0
        ;;
    *)
        echo "Unrecognized option: $1"
        usage
        exit 1
        ;;
  esac
done

tox -e flake8

if [[ ${HARBOR:-} = "" ]] && [[ ${DISTRIBUTION:-} = "" ]]; then
    HARBOR=1
    DISTRIBUTION=1
fi

if [[ ${HARBOR:-} != "" ]]; then
    echo "========================================================"
    echo "Start run validation test with Harbor registry server..."
    export REGISTRY_URL="${HARBOR_URL}"
    export REGISTRY_PASSWORD="${HARBOR_PASSWORD}"
    echo "REGISTRY_URL: ${REGISTRY_URL}"

    setup_harbor
    tox -e harbor || {
        delete_k3s_cluster
        echo "Hangar validation test with Harbor: FAILED"
        exit 1
    }
    delete_k3s_cluster
    echo "Hangar validation test with Harbor: Done"
fi

if [[ ${DISTRIBUTION:-} != "" ]]; then
    echo "========================================================"
    echo "Start run validation test with Distribution registry server..."
    export REGISTRY_URL="${DISTRIBUTION_URL}"
    # Distribution registry server can be login with any passwd by default
    # export REGISTRY_PASSWORD=""
    echo "REGISTRY_URL: ${REGISTRY_URL}"

    setup_distribution_registry
    tox -e distribution_registry || {
        delete_distribution_registry
        echo "Hangar validation test with Distribution registry: FAILED"
        exit 1
    }
    delete_distribution_registry
    echo "Hangar validation test with Distribution registry: Done"
fi
