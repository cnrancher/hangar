# image-tools

`image-tools` is a tool for mirror/copy multi-arch container images from the public registry to your own personal registry with manifest list support.

You can use `image-tools mirror` command to mirror images from source registry to your own registry.

Or you can use `image-tools save` and `image-tools load` commands to save the images from public registry to `tar.gz` tarball and then load it into your private registry in air-gap mode.

## Docs

For more detailed information about this project, please refer to the documents in [docs](docs/) folder.

> Simplified Chinese: [简体中文-使用文档](./docs/zh_CN/README.md)

## QuickStart

You can download the latest compiled binary from the [Release](/releases) page.

```sh
# Get help message
./image-tools -h

# Get help message of each command
./image-tools mirror -h
./image-tools save -h
......
```

## Build

```sh
# Ensure Go and make is installed
go version go1.19 linux/amd64

# Get help message
make help

# Build this project
make build

# Install executable binary into $GOPATH/bin
make install

# Run unit test
make test
```

## LICENSE

    Copyright 2022 SUSE Rancher

    Licensed under the Apache License, Version 2.0 (the "License");
    you may not use this file except in compliance with the License.
    You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

    Unless required by applicable law or agreed to in writing, software
    distributed under the License is distributed on an "AS IS" BASIS,
    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    See the License for the specific language governing permissions and
    limitations under the License.
