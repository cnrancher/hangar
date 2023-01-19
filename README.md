# image-tools

[![Build Status](https://drone-pandaria.cnrancher.com/api/badges/cnrancher/image-tools/status.svg?ref=refs/heads/main)](https://drone-pandaria.cnrancher.com/cnrancher/image-tools)

`image-tools` is a tool for mirroring/copying multi-arch container images from the public registry to your own personal registry with manifest list support.

You can use the `image-tools mirror` command to mirror images from the source registry to your own registry.

Or you can use the `image-tools save` and `image-tools load` commands to save the images from the public registry to the tar archive and then load them into your private registry in air-gap mode.

## Docs

For more detailed information about this project, please refer to the documents in [docs](docs/) folder.

> Simplified Chinese: [简体中文-使用文档](./docs/zh_CN/README.md)

## QuickStart

You can run `image-tools` from docker image.

```console
$ docker pull cnrancher/image-tools:${VERSION}

$ docker run cnrancher/image-tools:${VERSION} --help
Usage:	image-tools COMMAND [OPTIONS]
......

$ docker run --entrypoint bash -v $(pwd):/images -it cnrancher/image-tools:${VERSION}
```

You can download the latest compiled binary `image-tools-${OS}-${ARCH}-${VERSION}` from the [Release](https://github.com/cnrancher/image-tools/releases) page.

```sh
# Get help message
./image-tools -h

# Get help message for each command
./image-tools mirror -h
./image-tools save -h
......
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

    Copyright 2022-2023 Rancher Labs, Inc.

    Licensed under the Apache License, Version 2.0 (the "License");
    you may not use this file except in compliance with the License.
    You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

    Unless required by applicable law or agreed to in writing, software
    distributed under the License is distributed on an "AS IS" BASIS,
    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    See the License for the specific language governing permissions and
    limitations under the License.
