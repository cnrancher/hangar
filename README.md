# Hangar

[![Build Status](https://drone-pandaria.cnrancher.com/api/badges/cnrancher/hangar/status.svg?ref=refs/heads/main)](https://drone-pandaria.cnrancher.com/cnrancher/hangar)
[![Docker Pulls](https://img.shields.io/docker/pulls/cnrancher/hangar.svg)](https://store.docker.com/community/images/cnrancher/hangar)
[![Go Report Card](https://goreportcard.com/badge/github.com/cnrancher/hangar)](https://goreportcard.com/report/github.com/cnrancher/hangar)

Hangar is a tool for mirroring/copying multi-arch container images from the public registry to your registry with manifest list support, it also can generate an image list file according to Rancher KDM data and chart repositories for mirroring/saving images.

It provides the following subcommands:
- [mirror](docs/en_US/mirror.md): mirror images from the source registry to your own registry according to the image list file.
- [save](docs/en_US/save.md): download the image locally and generate a compressed package according to the image list file.
- [load](docs/en_US/load.md): load images from the compressed package and upload them to the personal registry.
- [convert-list](docs/en_US/convert-list.md): convert image list to 'mirror' format.
- [mirror-validate](docs/en_US/mirror-validate.md): validate the mirrored images.
- [load-validate](docs/en_US/load-validate.md): validate the loaded images.
- [generate-list](docs/en_US/generate-list.md): generate an image-list by Rancher KDM data and Chart repositories.

## Dependencies

Hangar uses [skopeo](https://github.com/containers/skopeo/blob/main/install.md) to copy container images and use [docker](https://docs.docker.com/get-docker/) client and [docker-buildx](https://docs.docker.com/build/install-buildx/) to build the manifest list.

## Docs

For more detailed usage information about this project, please refer to the documents in [docs](docs/) folder.

> [English](./docs/en_US/README.md) | [简体中文-使用文档](./docs/zh_CN/README.md)

## QuickStart

It's recommended to run `hangar` from the docker image without installing `skopeo` and `docker-buildx` dependencies manually.

```console
$ docker pull cnrancher/hangar:${VERSION}

$ docker run cnrancher/hangar:${VERSION} --help
Usage:	hangar COMMAND [OPTIONS]
......

$ docker run --entrypoint bash -v $(pwd):/images -it cnrancher/hangar:${VERSION}
```

----

Or you can download the latest compiled binary file `hangar-${OS}-${ARCH}-${VERSION}` from the [Release](https://github.com/cnrancher/hangar/releases) page.

```sh
# Download hangar binary file from GitHub Release
wget https://github.com/cnrancher/hangar/releases/download/<VERSION>/hangar-<OS>-<ARCH>-<VERSION> -O hangar
chmod +x ./hangar

# Get help message
./hangar -h
```

## Build

```sh
# Ensure Docker and make are installed

# Get help message
make help

# Build binary files into `build` folder
make build

# Run unit test
make test

# Delete binary files
make clean
```

## LICENSE

Copyright 2022-2023 [Rancher Labs, Inc](https://rancher.com).

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
