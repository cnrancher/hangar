#!/bin/bash

set -euo pipefail

on_error(){
    echo "error occured"
    FAILED=1
}

trap 'on_error' ERR

ls aaa
ls bbb
ls -al

if [[ $FAILED = 1 ]]; then
    echo "ERROR"
    exit 100
fi
