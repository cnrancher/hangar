# Validation Test

Validation tests for Hangar commands.

## Usage

### In Container

Use following commands to run hangar validation tests in a container.

1. Run `make build` to build hangar executable binary in container.
1. Run `make validation-test` to run validation tests for all hangar subcommands.

### Without Container

To run validation tests on your local machine:

1. Build hangar on your local machine by refer to [Building without a container](https://hangar.cnrancher.com/docs/dev/build#building-without-a-container)

1. Install python dependencies:

    ```sh
    cd test/
    pip install -r requirements.txt
    pip install tox
    ```

1. Run `scripts/registry.sh`, this will create a temporary *private registry server* for tests.

    ```sh
    ./scripts/registry.sh
    ```

1. To run specific test file:

    ```sh
    # Set REGISTRY_AUTH_FILE environment variable to avoid permission denied error during tests.
    export REGISTRY_AUTH_FILE="${HOME}/.config/containers/auth.json"

    # Specify the REGISTRY_URL environment variable manually.
    export REGISTRY_URL=127.0.0.1:5000

    cd suite/

    # Run specific test file.
    pytest -s test_help.py
    # Run specific test case.
    pytest -s test_help.py::test_help
    ```

1. Cleanup:

    - Run `scripts/clean.sh`.
    - Delete the `registry` directory.
