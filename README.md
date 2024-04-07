<div align="center">
  <h1>Hangar</h1>
  <p>
    <a href="https://build.opensuse.org/package/show/home:StarryWang/Hangar"><img alt="GitHub pre-release" src="https://build.opensuse.org/projects/home:StarryWang/packages/Hangar/badge.svg?type=default"></a>
    <a href="https://goreportcard.com/report/github.com/cnrancher/hangar"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/cnrancher/hangar"></a>
    <a href="https://github.com/cnrancher/hangar/releases"><img alt="GitHub release" src="https://img.shields.io/github/v/release/cnrancher/hangar?color=default&label=release&logo=github"></a>
    <a href="https://github.com/cnrancher/hangar/releases"><img alt="GitHub pre-release" src="https://img.shields.io/github/v/release/cnrancher/hangar?include_prereleases&label=pre-release&logo=github"></a>
    <img alt="License" src="https://img.shields.io/badge/License-Apache_2.0-blue.svg">
  </p>
</div>

> English | [简体中文](https://hangar.cnrancher.com/zh/)

Hangar is a command line utility for container images, it's main features are:
- Copy multi-platform container images between registry servers.
- Save and load multi-platform container images between archive files.
- Container image signing functions with sigstore key-pairs.
- Container image vulnerability scanning.

## Why use hangar?

- Hangar does not require any container runtime (daemon) to copy container images.
- Hangar is not restricted by the platform of the runtime system, it supports Linux/Unix systems.
- Hangar supports both [docker images](https://github.com/moby/docker-image-spec/blob/main/README.md) and [OCI images](https://github.com/opencontainers/image-spec).
- Hangar supports copy/save/load/sign/scan multi-platform images parallelly to increase speed.
- Hanagr is designed to save/load multi-platform container images with archve files in Air-Gapped environments.

## Getting started

The getting started instruction of Hangar is available in [documents](https://hangar.cnrancher.com/docs/v1.8/).

## Contributing

Hangar is open-source and any [issues](https://github.com/cnrancher/hangar/issues) or [pull requests](https://github.com/cnrancher/hangar/pulls) are welcomed if you have any suggestions while using Hangar.

## License

Copyright 2024 SUSE Rancher

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
