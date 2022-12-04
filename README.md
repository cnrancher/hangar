image-tools
===========

Mirror multi-arch container images from public registry to your personal registry with manifest list support.

Usage
-----

```sh
# go version
go version go1.19 linux/amd64

# Build image-tool
go build -o image-tools .

# Get help message
./image-tools -h

# Get help message of each command
./image-tools mirror -h
```

LICENSE
-------

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
