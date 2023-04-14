#!/bin/bash

set -uo pipefail

on_error(){
    echo "---------------"
    echo "| TEST FAILED |"
    echo "---------------"
    FAILED=1
}

cd $(dirname $0)/../
WORKINGDIR="$(pwd)/"

source ./scripts/env.sh

# help
echo "------ help ------"
pytest -s test_help.py

# version
echo "------ version ------"
pytest -s test_version.py

# mirror
echo "------ mirror | mirror-validate ------"
pytest -s test_mirror.py

# save
echo "------ save ------"
pytest -s test_save.py

# load
echo "------ load | load-validate ------"
pytest -s test_load.py

# sync
echo "------ sync | compress | decompress ------"
pytest -s test_sync_compress.py

# convert-list
echo "------ convert-list ------"
pytest -s test_convert_list.py

# generate-list
echo "------ generate-list ------"
pytest -s test_generate_list.py

if [[ ${FAILED:-} = 1 ]]; then
    echo "Some tests failed."
    exit 1
fi

echo "Done"
