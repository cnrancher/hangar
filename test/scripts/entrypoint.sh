#!/bin/bash

set -euo pipefail

cd $(dirname $0)/../
WORKINGDIR="$(pwd)/"

source scripts/env.sh

case ${1} in
    test_all)
        ${WORKINGDIR}/scripts/test-all.sh
        ;;
    test_mirror | test_mirror-validate)
        pytest -s test_mirror.py
        ;;
    test_save)
        pytest -s test_save.py
        ;;
    test_load | test_load-validate)
        pytest -s test_load.py
        ;;
    test_sync | test_compress | test_decompress)
        pytest -s test_sync_compress.py
        ;;
    test_convert-list)
        pytest -s test_convert_list.py
        ;;
    test_generate-list)
        pytest -s test_generate_list.py
        ;;
    test_version)
        pytest -s test_version.py
        ;;
    *)
        echo "Usage: $0 test_[COMMAND_NAME]"
        echo "Usage: $0 test_all"
        exit 1
        ;;
esac
