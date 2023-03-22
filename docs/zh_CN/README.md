# Hangar Usage (CN)

> 本仓库 `main` 分支的使用文档会随着版本更新而不断修改，若需要查看之前已发布的版本的使用文档，请切换至之前已发布的版本对应的 Tag:
> `https://github.com/cnrancher/hangar/blob/${TAG}/docs/zh_CN/README.md`

Hangar 是一个容器镜像拷贝以及生成 Rancher 离线安装时所需的镜像列表的工具，它支持下方 [Commands](#commands) 所介绍的命令，通过使用此工具，可简化离线环境安装 Rancher 时部署容器镜像到私有镜像仓库的步骤。

## COMMANDS

- [mirror](./mirror.md): 根据列表文件，将容器镜像拷贝至私有镜像仓库。
- [save](./save.md): 根据列表文件，将容器镜像保存至本地文件（压缩包或未压缩的文件夹中）。
- [load](./load.md): 将 [save](./save.md) 命令保存的镜像文件加载至私有仓库。
- [convert-list](./convert-list.md): 将 `rancher-images.txt` 格式的镜像列表转换为 [mirror](./mirror.md) 命令所使用的镜像列表格式。
- [mirror-validate](./mirror-validate.md): 对已 Mirror 的镜像校验。
- [load-validate](./load-validate.md): 对已 Load 的镜像校验。
- [sync](./sync.md): 将额外的镜像保存在 Save 命令创建的缓存文件夹中。
- [compress](./compress.md): 压缩镜像的缓存文件夹。
- [decompress](./decompress.md): 解压 Save/Compress 命令创建的镜像压缩文件。
- [generate-list](./generate-list.md): 根据 KDM 和 Chart 仓库生成一份镜像列表文件。

## 镜像仓库种类

Hangar 的 Mirror / Save / Load 相关命令支持的镜像仓库种类：

- Docker Hub
- Harbor V2
    > 此工具不支持 Harbor V1 仓库的 Mirror 和 Load 操作
- 公有云镜像平台，例如：腾讯云 TCR、阿里云 ACR

## 运行环境

推荐在容器中运行 Hangar 工具：`cnrancher/hangar:latest`；

有关 `hangar` 的 Docker 镜像的使用方式请参考 [docker-images 页面](./docker-images.md)。

----

若需要在系统中安装/运行此工具，请按照此部分安装 `skopeo`、`docker` 和 `docker-buildx` 第三方依赖：

1. Hangar 支持运行的系统为 Linux 或 macOS，架构为 amd64 或 arm64。
1. 安装 [skopeo](https://github.com/containers/skopeo/blob/main/install.md)。
1. 安装 `docker` 和 `docker-buildx` 插件。
1. 环境变量（可选）：

    在执行 Mirror / Save / Load 命令时，Hangar 支持读取系统中以下的环境变量，适用于在 CI 中自动执行镜像 Mirror / Save 等操作。
    - `SOURCE_USERNAME`: 源 Registry 用户名
    - `SOURCE_PASSWORD`: 源 Registry 密码
    - `SOURCE_REGISTRY`: 源 Registry 地址
    - `DEST_USERNAME`: 目标 Registry 用户名
    - `DEST_PASSWORD`: 目标 Registry 密码
    - `DEST_REGISTRY`: 目标 Registry 地址

    以 `SOURCE_` 开头的环境变量表示待 Mirror / Save 的源镜像的 Registry 的用户名、密码、URL；<br>
    以 `DEST_` 开头的环境变量表示被 Mirror / Load 的目标镜像的 Registry 的用户名、密码、URL。

    > Hangar 除了通过环境变量获取 Registry 的用户名和密码外，还会尝试从 `~/.docker/config.json` 文件获取 Registry 的用户名和密码，

    若未获取到用户名密码，那么 Hangar 会提示手动输入用户名和密码。

1. 在使用自建 SSL Certificate 时，请参照 [自建 SSL Certificate](./self-signed-ssl.md) 进行配置。

## 常见问题

常见报错信息及解释：[常见问题](./questions.md)

## Build

> 可在本仓库的 [Release 页面](https://github.com/cnrancher/hangar/releases) 获取已构建的稳定版本。

构建可执行文件请参考 [build.md](./build.md)
