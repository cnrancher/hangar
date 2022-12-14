# image-tools usage (CN)

```
./image-tools COMMAND OPTIONS
```

## 运行环境

1. Linux 或 macOS 系统，架构为 amd64 或 arm64
1. 在 Release 页面下载构建物，使用 `unxz` 解压获取编译后的可执行文件

    ```console
    $ unxz image-tools-${OS}-${ARCH}-${VERSION}.xz
    ```

1. 确保 [skopeo](https://github.com/containers/skopeo) 已安装
1. 确保 `docker` 和 `docker-buildx` 插件已安装。

    （`docker` 和 `docker-buildx` 可使用最新版本）

1. 设定环境变量（可选）：
    以下环境变量在 `mirror` 或 `load` 时可设定目标 registry 的用户名、密码和 URL。
    - `DOCKER_USERNAME`: 目标 registry 用户名
    - `DOCKER_PASSWORD`: 目标 registry 密码
    - `DOCKER_REGISTRY`: 目标 registry 地址

    若未设定环境变量，可在执行该工具时手动输入用户名和密码。
1. 在使用自建 SSL Certificate 时，请参照 [自建 SSL Certificate](./self-signed-ssl.md) 进行配置。

## COMMANDS

- [mirror](./mirror.md): 根据列表文件，将镜像拷贝至私有镜像仓库。
- [save](./save.md): 根据列表文件，将镜像下载至本地，生成 `tar.gz` 压缩包。
- [load](./load.md): （离线环境）读取压缩包，将压缩包内镜像上传至私有仓库。
- [convert-list](./convert-list.md) 转换镜像列表格式。

## Build

构建可执行文件：[build.md](./build.md)
