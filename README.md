# Hangar

<div align="center">
  <p>
    <a href="https://drone-pandaria.cnrancher.com/cnrancher/hangar"><img alt="Build Status" src="http://drone-pandaria.cnrancher.com/api/badges/cnrancher/hangar/status.svg"></a>
    <a href="https://goreportcard.com/report/github.com/cnrancher/hangar"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/cnrancher/hangar"></a>
    <a href="https://github.com/cnrancher/hangar/releases"><img alt="GitHub release" src="https://img.shields.io/github/v/release/cnrancher/hangar?color=default&label=release&logo=github"></a>
    <img alt="License" src="https://img.shields.io/badge/License-Apache_2.0-blue.svg">
  </p>
</div>

> English | [简体中文](https://hangar.cnrancher.com/zh/)

Hangar is a **simple** and **easy-to-use** command line utility for mirroring multi-architecture & multi-platform container images between container image registries. Aiming to simplify the process of copying container images between image registries.

- Hangar allows to copy container images without container runtime (daemon).
- Not restricted by the architecture and OS of the runtime system, it supports running on Linux/Unix systems.
- Supports both docker images and [OCI images](https://github.com/opencontainers/image-spec).
- Hangar supports to copy container images parallelly to improve performance.
- Save and load images with archive files to allow the setup of registry server in Air-Gapped installation.

Hangar provides following functions：

- Mirror container images between image registries (see [mirror](https://hangar.cnrancher.com/docs/mirror/mirror) subcommand).
- Save container images into an archive file, and then upload them to the image registry server (see [save](https://hangar.cnrancher.com/docs/save/save) and [load](https://hangar.cnrancher.com/docs/load/load) subcommands). Designed to use for Air-Gapped (offline) installation.
- Validate commands to verify that the container images were copied correctly (see [validate](https://hangar.cnrancher.com/docs/advanced-usage/validate) subcommands).
- Other advanced commands for image list files and archive files (see [advanced usage](https://hangar.cnrancher.com/docs/advanced-usage/)).

## Documents

The detailed usage of Hangar and getting started instruction is available in [hangar.cnrancher.com](https://hangar.cnrancher.com).

## Dependencies

Starting from `v1.7.0`, Hangar has removed all third-party executable binary dependencies to improve the speed of container image copying and reduce performance consumption.

## Contributing

If you encounter any issues or have suggestions for improvement while using Hangar, feel free to open an [issue](https://github.com/cnrancher/hangar/issues) or [pull request](https://github.com/cnrancher/hangar/pulls). Your contributions are welcomed!

## License

Copyright 2023 SUSE Rancher

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
