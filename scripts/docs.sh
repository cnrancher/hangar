#!/bin/bash

set -euo pipefail

cd $(dirname $0)/../

MAN_DIR="dist/man1"
mkdir -p $MAN_DIR

echo "Generating manpage to '$MAN_DIR'..."
go run docs/main.go $(pwd)/$MAN_DIR

echo "docs: Done"
