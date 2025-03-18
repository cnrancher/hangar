<div align="center">
  <h1>Hangar</h1>
  <p>
    <a href="https://build.opensuse.org/package/show/home:StarryWang/Hangar"><img src="https://build.opensuse.org/projects/home:StarryWang/packages/Hangar/badge.svg?type=default"></a>
    <a href="https://aur.archlinux.org/packages/hangar"><img src="https://img.shields.io/aur/version/hangar"></a>
    <a href="https://goreportcard.com/report/github.com/cnrancher/hangar"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/cnrancher/hangar"></a>
    <a href="https://github.com/cnrancher/hangar/releases"><img alt="GitHub release" src="https://img.shields.io/github/v/release/cnrancher/hangar?color=default&label=release&logo=github"></a>
    <a href="https://github.com/cnrancher/hangar/releases"><img alt="GitHub pre-release" src="https://img.shields.io/github/v/release/cnrancher/hangar?include_prereleases&label=pre-release&logo=github"></a>
    <img alt="License" src="https://img.shields.io/badge/License-Apache_2.0-blue.svg">
  </p>
</div>

> English | [简体中文](https://hangar.cnrancher.com/zh/)

Hangar is a command line utility for container images with the following features:

- Multi-platform container images.
- Copy container images between registry servers.
- Export container images as archive files and import them into image repositories.
- Sign container images with sigstore key-pairs.
- Scan container image vulnerabilities.

## Why use hangar?

- Hangar does not require any container runtime (daemon) to copy container images.
- Hangar is cross-platform and works in all Unix-like operating systems.
- Hangar supports both [docker images](https://github.com/moby/docker-image-spec/blob/main/README.md) and [OCI images](https://github.com/opencontainers/image-spec).
- Hangar supports copying/saving/loading/signing/scanning images in parallel to increase speed.
- Hangar is designed to export container images as archive files and import them into image repositories in Air-Gapped environments.

## Getting started

For documentation, visit the [Hangar Documentation](https://hangar.cnrancher.com/docs/v1.9).

## Contributing

Hangar is open-source and any [issues](https://github.com/cnrancher/hangar/issues) or [pull requests](https://github.com/cnrancher/hangar/pulls) are welcomed if you have any suggestions while using Hangar.

## License

Copyright 2025 SUSE Rancher

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
