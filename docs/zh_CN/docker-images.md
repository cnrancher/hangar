# docker-images

> `hangar` 从 `v1.3.0` 版本开始支持 Docker 镜像。

Docker 镜像支持 `amd64` 和 `arm64` 架构。

```sh
# 获取镜像
docker pull cnrancher/hangar:${VERSION}

# 获取帮助信息
## 默认情况下 entrypoint 为 hangar 可执行文件
docker run cnrancher/hangar:${VERSION} --help
```

设定 `entrypoint` 为 `bash`, 将本地目录挂载到容器中，可在容器内执行 Mirror / Load / Save。

```console
$ docker run --entrypoint bash -v $(pwd):/images -it cnrancher/hangar:${VERSION}
a455e1202691:/images # hangar -h
Usage:	hangar COMMAND [OPTIONS]
......
```

## 在 CI 中运行 Mirror

在 CI Pipeline 中可自动运行 Mirror 命令，可通过设定以下环境变量指定源镜像 Registry 和目标 Registry 以及用户名密码。

- `SOURCE_USERNAME`: 源 Registry 用户名
- `SOURCE_PASSWORD`: 源 Registry 密码
- `SOURCE_REGISTRY`: 源 Registry 地址
- `DEST_USERNAME`: 目标 Registry 用户名
- `DEST_PASSWORD`: 目标 Registry 密码
- `DEST_REGISTRY`: 目标 Registry 地址

----

Example:

```bash
#!/bin/bash

docker run -v $(pwd):/images \
    -e SOURCE_REGISTRY="" \
    -e SOURCE_USERNAME="" \
    -e SOURCE_PASSWORD="" \
    -e DEST_REGISTRY="" \
    -e DEST_USERNAME="" \
    -e DEST_PASSWORD="" \
    cnrancher/hangar:${VERSION} mirror \
    -f /images/list.txt \
    -o /images/mirror-failed.txt

# check mirror-failed.txt
cat mirror-failed.txt
```
