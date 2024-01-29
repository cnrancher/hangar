# Hangar

[![Build Status](https://drone-pandaria.cnrancher.com/api/badges/cnrancher/hangar/status.svg?ref=refs/heads/main)](https://drone-pandaria.cnrancher.com/cnrancher/hangar)
[![Docker Pulls](https://img.shields.io/docker/pulls/cnrancher/hangar.svg)](https://store.docker.com/community/images/cnrancher/hangar)
[![Go Report Card](https://goreportcard.com/badge/github.com/cnrancher/hangar)](https://goreportcard.com/report/github.com/cnrancher/hangar)

Hangar is a tool for mirroring/copying multi-arch container images from the public registry to your registry with manifest list support, it also can generate an image list file according to Rancher KDM data and chart repositories for mirroring/saving images.

It provides the following subcommands:
- [mirror](https://hangar.cnrancher.com/docs/v1.6/mirror/mirror): Mirror the container image to the private registry.
- [save](https://hangar.cnrancher.com/docs/v1.6/save/save): Download the container image to the local file and generate a compressed package.
- [load](https://hangar.cnrancher.com/docs/v1.6/load/load): Load the file created by [save](./docs/zh_CN/save.md) command onto the private registry.
- [convert-list](https://hangar.cnrancher.com/docs/v1.6/advanced/convert-list): Convert image list from `rancher-images.txt` to format used by [mirror](https://hangar.cnrancher.com/docs/v1.6/mirror/mirror) command.
- [mirror-validate](https://hangar.cnrancher.com/docs/v1.6/mirror/validate): Validate the mirrored images.
- [load-validate](https://hangar.cnrancher.com/docs/v1.6/load/validate): Validate the loaded images.
- [sync](https://hangar.cnrancher.com/docs/v1.6/advanced/sync): Sync extra images into image cache folder.
- [compress](https://hangar.cnrancher.com/docs/v1.6/advanced/compress): Compress the image cache folder.
- [decompress](https://hangar.cnrancher.com/docs/v1.6/advanced/decompress): Decompress tarball created by [save](./save.md) command.
- [generate-list](https://hangar.cnrancher.com/docs/v1.6/advanced/generate-list): Generate an image-list by KDM data and Chart repositories.

## Docs

For more detailed usage information about this project, please refer to the hangar documents website <https://hangar.cnrancher.com>.

## QuickStart

It's recommended to run `hangar` from the docker image without installing `skopeo` dependency manually, see [docker-images.md](https://hangar.cnrancher.com/docs/v1.6/docker-images).

```console
$ docker pull cnrancher/hangar:${VERSION}

$ docker run cnrancher/hangar:${VERSION} hangar --help
Usage:	hangar COMMAND [OPTIONS]
......

$ docker run -v $(pwd):/hangar -it cnrancher/hangar:${VERSION}
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

See [build.md](https://hangar.cnrancher.com/docs/v1.6/dev/build) document.

## LICENSE

Copyright 2024 SUSE Rancher.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
