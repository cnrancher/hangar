# Hangar usage (EN)

> The usage documentation of the `main` branch of this tool will be continuously modified with version updates. If you need to view the usage documentation of a previously released version, please switch to the Tag corresponding of the previously released version:
> `https://github.com/cnrancher/hangar/blob/${TAG}/docs/en_US/README.md`

```
./hangar COMMAND OPTIONS
```

## Supported Registries

- Docker Hub
- Harbor V2
> This tool does not support the Mirror and Load operations of the Harbor V1 registry
- Public cloud mirroring platforms, such as: Tencent Cloud TCR, Alibaba Cloud ACR

## Environment

This tool supports running in a container. Please refer to [docker-images.md](./docker-images.md) for how to use `hangar` Docker images.

To run this tool locally, please install `skopeo`, `docker` and `docker-buildx` as follows:

1. Linux or macOS system, the architecture is amd64 or arm64
2. Make sure [skopeo](https://github.com/containers/skopeo/blob/main/install.md) is installed

    > skopeo >= `0.1.40`

    openEuler:

    ```sh
    sudo yum install skopeo
    ```

    Ubuntu 20.04 compiled executables:
    - [skopeo-1.9.3-amd64](https://starry-public-files.s3.ap-northeast-1.amazonaws.com/skopeo/amd64/1.9.3/skopeo)
    - [skopeo-1.9.3-arm64](https://starry-public-files.s3.ap-northeast-1.amazonaws.com/skopeo/arm64/1.9.3/skopeo)

    ```sh
    # Ubuntu 20.10 and newer
    sudo apt-get -y update
    sudo apt-get -y install skopeo
    ```

    macOS:

    ```sh
    brew install skopeo
    ```

3. Make sure `docker` and `docker-buildx` are installed.

    (`docker` and `docker-buildx` can use the latest version)

    - openEuler 22.03-LTS can use [this script](https://github.com/cnrancher/euler-packer/blob/main/scripts/others/install-docker.sh) to install `docker` and `docker-buildx`.
    - For other systems, please refer to [Docker official website](https://docs.docker.com/get-docker/) and [Docker Buildx](https://docs.docker.com/build/install-buildx/) page installation .

4. Set environment variables (optional):

    The following environment variables can set the username, password and URL of the source/target Registry when executing this tool, which is used for automatic Mirroring in CI scenarios.
    - `SOURCE_USERNAME`: Source Registry username
    - `SOURCE_PASSWORD`: Source Registry password
    - `SOURCE_REGISTRY`: Source Registry address
    - `DEST_USERNAME`: Destination Registry username
    - `DEST_PASSWORD`: Destination Registry password
    - `DEST_REGISTRY`: Destination Registry address

    In addition, this tool will obtain the Registry of the target image from the image list during Mirror / Load and execute `docker login` on it.

    If the image to be Mirrored / Saved is a private image, you can execute `docker login` on the Registry of the source image by setting `SOURCE_*` environment variables.

    In addition to obtaining the username and password of the Registry through environment variables, this tool will also try to obtain the username and password of the Registry from the `~/.docker/config.json` file.
    If the user name and password are not obtained, the tool will prompt to manually enter the user name and password.

1. When using a self-signed SSL Certificate, please refer to [Self-signed SSL Certificate](./self-signed-ssl.md) for configuration.

## COMMANDS

- [mirror](./mirror.md): According to the list file, copy the image to the private registry.
- [save](./save.md): According to the list file, download the image to the local and generate a compressed package.
- [load](./load.md): (Air-gap mode) read the compressed package and upload the image to a private registry.
- [convert-list](./convert-list.md): Convert image list to 'mirror' format.
- [mirror-validate](./mirror-validate.md): Validate the mirrored image.
- [load-validate](./load-validate.md): Validate the loaded image.
- [generate-list](./generate-list.md): Generate an image-list by KDM data and Chart repositories.

## Common problems

Common error messages and explanations: [FAQ](./questions.md)

## Principle

This tool uses the `skopeo` command to copy the image to the target mirror server or a local folder, and uses `docker-buildx` to create a Manifest list for the target mirror server.

This tool only needs `skopeo`, `docker` client and `docker-buildx` plugin, and not the Docker daemon.

## Build

> The stable release can be obtained from the [Releases page](https://github.com/cnrancher/hangar/releases).

Build executable: [build.md](./build.md)
