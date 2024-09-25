#!/bin/bash

# Launch K3s cluster in docker and install harbor (insecure tls certificate)
# URL: https://127.0.0.1, https://localhost
function setup_harbor() {
    set -euo pipefail

    K3S_CLUSTER_NAME=${K3S_CLUSTER_NAME}
    HARBOR_PORT=${HARBOR_PORT:-443}
    HARBOR_HELM_VERSION=${HARBOR_HELM_VERSION}
    REGISTRY_URL=${REGISTRY_URL}
    REGISTRY_PASSWORD=${REGISTRY_PASSWORD}

    type k3d > /dev/null
    type helm > /dev/null
    type kubectl > /dev/null
    type docker > /dev/null

    echo "Launching K3s cluster..."
    if k3d cluster ls --no-headers | grep -q ${K3S_CLUSTER_NAME}; then
        CLUSTERS=$(k3d cluster ls --no-headers | cut -d ' ' -f 1)
        for c in ${CLUSTERS}; do
            k3d cluster delete ${c} || true
            sleep 1
        done
    fi
    k3d cluster create -p ${HARBOR_PORT}:${HARBOR_PORT} ${K3S_CLUSTER_NAME}
    sleep 3 # Add some timeout

    echo "Install harbor helm chart.."
    rm -r harbor || true
    if [[ ! -e  harbor.zip ]]; then
        wget https://github.com/goharbor/harbor-helm/archive/refs/tags/${HARBOR_HELM_VERSION}.zip -O harbor.zip
        unzip harbor.zip > /dev/null && mv harbor-helm-${HARBOR_HELM_VERSION#v} harbor && cd harbor
        helm install \
            --set expose.type=ingress \
            --set expose.tls.enabled=true \
            --set expose.tls.certSource=auto \
            --set expose.tls.auto.commonName="tls-harbor" \
            --set expose.ingress.hosts.core="${REGISTRY_URL}" \
            --set externalURL="https://${REGISTRY_URL}" \
            --set harborAdminPassword="${REGISTRY_PASSWORD}" \
            --set secretKey="AAA-X-secure-Cey" \
            harbor .
    fi

    i=0
    echo "Waiting for harbor server initialized..."
    while [[ $(kubectl get deployments.apps harbor-core -o json | jq '.status.availableReplicas' || true) != "1"  ]] && [[ $i -lt  30 ]]; do
        echo "Waiting for harbor core service..."
        kubectl get deployments.apps harbor-core --no-headers
        echo "-----------------------------------"
        sleep 10
        i=$(( $i + 1 ))
    done
    if [[ $i -gt 30 ]]; then # Waiting for 5 minutes.
        echo "Timeout waiting for harbor core service, cleanup resources"
        k3d cluster delete ${K3S_CLUSTER_NAME} || true
        exit 1
    fi
}

function delete_k3s_cluster() {
    set -euo pipefail

    K3S_CLUSTER_NAME=${K3S_CLUSTER_NAME}

    k3d cluster delete ${K3S_CLUSTER_NAME} || true
}
