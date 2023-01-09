# image-tools usage (CN)

> 本仓库 `main` 分支的使用文档会随着版本更新而不断修改，若需要查看之前已发布的版本的使用文档，请切换至之前已发布的版本对应的 Tag:
> `https://github.com/cnrancher/image-tools/blob/${TAG}/docs/zh_CN/README.md`

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

    > skopeo 版本需大于等于 `0.1.40`

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

    以下环境变量在执行此工具时可设定源/目标 Registry 的用户名、密码和 URL，用于在 CI 场景中自动 Mirror 镜像。
    - `SOURCE_USERNAME`: 源 Registry 用户名
    - `SOURCE_PASSWORD`: 源 Registry 密码
    - `SOURCE_REGISTRY`: 源 Registry 地址
    - `DEST_USERNAME`: 目标 Registry 用户名
    - `DEST_PASSWORD`: 目标 Registry 密码
    - `DEST_REGISTRY`: 目标 Registry 地址

    除此之外本工具会在 Mirror / Load 时从镜像列表中获取目标镜像的 Registry 并对其执行 `docker login`。

    若待 Mirror / Save 的镜像为私有镜像，可通过设定 `SOURCE_*` 环境变量，对源镜像的 Registry 执行 `docker login`。

    本工具除了通过环境变量获取 Registry 的用户名和密码外，还会尝试从 `~/.docker/config.json` 文件中获取 Registry 的用户名和密码，
    若未获取到用户名密码，那么本工具会提示手动输入用户名和密码。

1. 在使用自建 SSL Certificate 时，请参照 [自建 SSL Certificate](./self-signed-ssl.md) 进行配置。

## COMMANDS

- [mirror](./mirror.md): 根据列表文件，将镜像拷贝至私有镜像仓库。
- [save](./save.md): 根据列表文件，将镜像下载至本地，生成压缩包。
- [load](./load.md): （离线环境）读取压缩包，将压缩包内镜像上传至私有仓库。
- [convert-list](./convert-list.md): 转换镜像列表格式。
- [mirror-validate](./mirror-validate.md): 对已 Mirror 的镜像校验。
- [load-validate](./load-validate.md): 对已 Load 的镜像校验。

## 常见问题

常见报错信息及解释：[常见问题](./questions.md)

## 原理

本工具使用 `skopeo` 命令拷贝镜像至目标镜像服务器或本地文件夹中，并使用 `docker-buildx` 为目标镜像服务器创建 Manifest 列表。

本工具仅需要 `skopeo`，`docker` 客户端以及 `docker-buildx` 插件，不需要 Docker Daemon。

## Build

> 可在本仓库的 [Release 页面](https://github.com/cnrancher/image-tools/releases) 获取已构建的稳定版本。

构建可执行文件：[build.md](./build.md)
