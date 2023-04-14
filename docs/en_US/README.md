# Hangar usage (EN)

> The documentation of the `main` will be continuously modified. You can adjust the TAG to view the preivious documents:
> `https://github.com/cnrancher/hangar/blob/${TAG}/docs/en_US/README.md`

## COMMANDS

- [mirror](./mirror.md): Mirror the container image to the private registry.
- [save](./save.md): Download the container image to the local file and generate a compressed package.
- [load](./load.md): Load the file created by [save](./save.md) command onto the private registry.
- [convert-list](./convert-list.md): Convert image list from `rancher-images.txt` to format used by [mirror](./mirror.md) command.
- [mirror-validate](./mirror-validate.md): Validate the mirrored image.
- [load-validate](./load-validate.md): Validate the loaded image.
- [sync](./sync.md): Sync extra images into image cache folder.
- [compress](./compress.md): Compress the image cache folder.
- [decompress](./decompress.md): Decompress tarball created by [save](./save.md) command.
- [generate-list](./generate-list.md): Generate an image-list by KDM data and Chart repositories.

## Supported Registries

- Docker Hub
- Harbor V2
    > Hangar does not support Harbor V1 registry
- Public cloud platforms: Tencent Cloud TCR, Alibaba Cloud ACR, etc

## Environment

Hangar supports running in container, see [docker-images.md](./docker-images.md).

To install hangar in your system, please install `skopeo`, `docker` and `docker-buildx` dependencies:

1. Linux / macOS, architecture amd64 / arm64
1. Install [skopeo](https://github.com/containers/skopeo/blob/main/install.md)
1. Make sure `docker` and `docker-buildx` are installed.
1. Set environment variables (optional):

    When running Mirror / Save / Load, following environment variables sets the username, password and URL of the source / destination registry
    (used in CI scenario).

    - `SOURCE_USERNAME`: Source registry username
    - `SOURCE_PASSWORD`: Source registry password
    - `SOURCE_REGISTRY`: Source registry address
    - `DEST_USERNAME`: Destination registry username
    - `DEST_PASSWORD`: Destination registry password
    - `DEST_REGISTRY`: Destination registry address

    > If not specifying username / password, hangar will also try to obtain the username and password of the registry from the `~/.docker/config.json` file.

    If the user name and password are not set, hangar will prompt to enter the user name and password manually.

1. When using a self-signed SSL Certificate, please refer to [Self-signed SSL Certificate](./self-signed-ssl.md) for configuration.

## Tests

See [test docs](./test.md).

## FAQ

[FAQ](./questions.md)

## Build

> The stable release can be obtained from the [Releases page](https://github.com/cnrancher/hangar/releases).

Build executable binaries: [build.md](./build.md)
