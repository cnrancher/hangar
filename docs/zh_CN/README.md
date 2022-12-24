# image-tools usage (CN)

```
./image-tools COMMAND OPTIONS
```

## 镜像仓库种类

- Docker Hub
- Harbor V2
    > 此工具不支持 Harbor V1 仓库的 Mirror 和 Load 操作
- 公有云镜像平台，例如：腾讯云 TCR、阿里云 ACR

## 运行环境

1. Linux 或 macOS 系统，架构为 amd64 或 arm64
1. 确保 [skopeo](https://github.com/containers/skopeo/blob/main/install.md) 已安装

    openEuler:

    ```sh
    sudo yum install skopeo
    ```

    Ubuntu 20.04 可下载已编译的可执行文件：
    - [skopeo-1.9.3-amd64](https://starry-public-files.s3.ap-northeast-1.amazonaws.com/skopeo/amd64/1.9.3/skopeo)
    - [skopeo-1.9.3-arm64](https://starry-public-files.s3.ap-northeast-1.amazonaws.com/skopeo/arm64/1.9.3/skopeo)

    ``` sh
    # Ubuntu 20.10 and newer
    sudo apt-get -y update
    sudo apt-get -y install skopeo
    ```

    macOS:

    ```sh
    brew install skopeo
    ```

1. 确保 `docker` 和 `docker-buildx` 插件已安装。

    （`docker` 和 `docker-buildx` 可使用最新版本）

    - openEuler 22.03-LTS 可使用 [此脚本](https://github.com/cnrancher/euler-packer/blob/main/scripts/others/install-docker.sh) 一键安装 `docker` 和 `docker-buildx`。
    - 其他系统请参照 [Docker 官网](https://docs.docker.com/get-docker/) 和 [Docker Buildx](https://docs.docker.com/build/install-buildx/) 页面安装。

1. 设定环境变量（可选）：

    以下环境变量在 `mirror` 或 `load` 时可设定目标 registry 的用户名、密码和 URL。
    - `DOCKER_USERNAME`: 目标 registry 用户名
    - `DOCKER_PASSWORD`: 目标 registry 密码
    - `DOCKER_REGISTRY`: 目标 registry 地址

    本工具获取 `docker login` 的 registry 的方式为：
    1. 若 Mirror 或 Load 时使用 `-d` 参数设定了目标 registry，那么使用此 registry 登录
    1. 若未使用 `-d` 参数，尝试读取 `DOCKER_REGISTRY` 环境变量
    1. 若未设定环境变量，那么设定 `docker login` 的 registry 为 Docker Hub 的 `docker.io`

    本工具 **不会** 从镜像列表中获取 `docker login` 的目标 registry，若镜像列表中目标镜像的 registry 不为 `docker.io`，
    请使用 `-d` 参数或 `DOCKER_REGISTRY` 环境变量显示的指明目标 registry，否则可能会导致 Mirror 或 Load 执行失败。

    本工具获取 `docker login` 的的用户名密码的顺序为：
    1. 首先尝试获取 `DOCKER_USERNAME` 和 `DOCKER_PASSWORD` 环境变量
    1. 若未设定环境变量，则尝试从 `~/.docker/config.json` 中获取已保存的用户名和密码信息
    1. 若仍未找到用户名和密码信息，那么提示手动输入用户名和密码

1. 在使用自建 SSL Certificate 时，请参照 [自建 SSL Certificate](./self-signed-ssl.md) 进行配置。

## COMMANDS

- [mirror](./mirror.md): 根据列表文件，将镜像拷贝至私有镜像仓库。
- [save](./save.md): 根据列表文件，将镜像下载至本地，生成 `tar.gz` 压缩包。
- [load](./load.md): （离线环境）读取压缩包，将压缩包内镜像上传至私有仓库。
- [convert-list](./convert-list.md) 转换镜像列表格式。

## Build

构建可执行文件：[build.md](./build.md)
