# image-tools usage (CN)

```
./image-tools COMMAND OPTIONS
```

## Dependency

1. OS: Linux / macOS;
    Arch: amd64 / arm64
1. Ensure [skopeo](https://github.com/containers/skopeo) is installed on your system.
    > This image-tool can download `skopeo` automatically if your system is Linux when `skopeo` is not installed on your system.
1. Ensure the latest version `docker` and `docker-buildx` are installed.
1. Setup environment variables (Optional)

    The following environment variables are used to specify the username, password and registry URL when executing `mirror` or `load` command.

    - `DOCKER_USERNAME`: Destination registry login username
    - `DOCKER_PASSWORD`: Destination registry login password
    - `DOCKER_REGISTRY`: Destination registry URL

    You can input username and password manually if these environment variables are not specified when running this tool.

## COMMANDS

- [mirror](./mirror.md): mirror images by image-list txt file.
- [save](./save.md): Save images into `tar.gz` by image-list txt file。
- [load](./load.md): (Air-Gap) Load `tar.gz` archive, load images to private registry.
- [convert-image](./convert-list.md): Convert image list format.

## Build

[build.md](./build.md)
