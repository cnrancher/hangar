#!/usr/bin/env bash

set -euo pipefail

if [[ $# -gt 1 ]]; then
    exec "$@"
else
    bash
fi
