# Validation Test

Validation tests for Hangar commands.

## Usage

### In Container

Use following commands to run hangar validation tests in docker container.

1. Run `make build` to build hangar executable binary in container.
1. Run `make validation-test` to run validation tests for all hangar subcommands.
    This will seutp a temporary k3s cluster and install harbor for test.

### Without Container

To run validation tests on your local machine:

1. Build hangar on your local machine by refer to [Building without a container](https://hangar.cnrancher.com/docs/dev/build#building-without-a-container)

1. Create virtual python environment by [uv]() and install python dependencies:

    ```sh
    cd test/

    uv venv
    uv pip install -r requirements.txt
    uv pip install tox
    ```

1. You need to prepare a [Harbor](https://goharbor.io) or [distribution](https://distribution.github.io/distribution/) registry server manually to run validation tests manually.

1. To run specific test file:

    ```sh
    # Set REGISTRY_AUTH_FILE environment variable to avoid permission denied error during tests.
    export REGISTRY_AUTH_FILE="${HOME}/.config/containers/auth.json"

    # Ensure the `default-docker.use-sigstore-attachments` is true.
    vim /etc/containers/registries.d/default.yaml

    # Specify the REGISTRY_URL environment variable manually.
    export REGISTRY_URL=127.0.0.1:5000
    # Set REGISTRY_PASSWORD if needed.
    export REGISTRY_PASSWORD="Harbor123!@#"

    # Run specific test file.
    uv run pytest -s suite/test_help.py
    # Run specific test case.
    uv run pytest -s suite/test_help.py::test_help
    ```

1. Cleanup:

    - Run `scripts/clean.sh`.
