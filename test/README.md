# Validation Test

Validation test for Hangar commands.

## Usage

1. Run `make build` in previous directory to build executable before running validation tests.
1. Ensure `pytest` installed.
1. Launch a Harbor v2 for testing purpose, configure username/password
in `./scripts/env.sh`.
    ```sh
    export SOURCE_REGISTRY="" # set to empty
    export SOURCE_USERNAME="" # username of dockerhub (optional)
    export SOURCE_PASSWORD="" # password of dockerhub (optional)

    export DEST_REGISTRY="" # harbor registry url
    export DEST_USERNAME="" # harbor username
    export DEST_PASSWORD="" # harbor password
    ```

    > If the Harbor v2 uses HTTP or insecure TLS certificate,
    > set `export TEST_SKIP_TLS="true"` to skip tls verify.
1. Run tests for subcommand:
    Run `pytest -s ./test_mirror.py` before test `save/load/compress/decompress` commands.
    ```console
    $ pytest -s ./test_mirror.py
    $ pytest -s ./test_[COMMAND].py
    ```
1. Run all tests:
    ```console
    $ ./scripts/test-all.sh
    ```
1. Clean generated files:
    ```console
    $ ./scripts/clean.sh
    ```
