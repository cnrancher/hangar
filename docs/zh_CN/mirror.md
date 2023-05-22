# Mirror

## 镜像列表格式

> `mirror` 命令和 `mirror-validate` 命令所输入的镜像列表格式与 `rancher-images.txt` 格式不一致，若需要将 `rancher-images.txt` 转换为 Mirror 命令所使用的镜像列表格式，请使用 [convert-list](./convert-list.md) 命令。

每一行包含 **“源镜像 目标镜像 TAG”**，以空格分隔，例如：

```txt
# <SOURCE> <DEST> <TAG>
docker.io/hello-world private.io/library/hello-world latest
```

源镜像和目标镜像可以为不包含 registry 前缀的镜像，例如：

```txt
# <SOURCE> <DEST> <TAG>
hello-world library/hello-world latest
```

> 若该行以 `#` 或 `//` 开头，那么这一行将被视为注释

## QuickStart

将 `image-list.txt` 列表中的所有镜像执行 Mirror，使用 `-f` 参数指定镜像列表名称，`-d` 指定目标 registry

```sh
hangar mirror -f ./image-list.txt -d <DEST_REGISTRY_URL>
```

### Harbor V2

若目标镜像仓库类型为 Harbor V2，可使用 `--repo-type=harbor` 参数，自动为 Harbor V2 仓库创建 Project。

> 若 Harbor V2 为 HTTP，还需要添加 `--harbor-https=false` 参数。

除此之外若镜像列表中的目标镜像不包含 `Project` （例如 `mysql:8.0`, `busybox:latest`），那么在 mirror 过程中会自动为其添加 `library` Project 前缀（`library/mysql:8.0`，`library/busybox:latest`）。

可使用 `--default-project=library` 参数设定添加 Project 的名称 （默认为 `library`）。

## Parameters

命令行参数：

```sh
# 使用 -f, --file 参数指定镜像列表文件
hangar mirror -f ./list.txt

# 使用 -d, --destination 参数，可以在不修改镜像列表的情况下，指定目标镜像的 registry
# 优先级为：-d 参数 > DOCKER_REGISTRY 环境变量 > 镜像列表中已写好的 registry
hangar mirror -f ./list.txt -d private.registry.io

# 使用 -s, --source 参数，可以在不修改镜像列表的情况下，指定源镜像的 registry
hangar mirror -f ./list.txt -s docker.io

# 使用 -a, --arch 参数，设定拷贝镜像的架构（以逗号分隔）
# 默认为 amd64,arm64
hangar mirror -f ./list.txt -a amd64,arm64

# 使用 --no-arch-failed=false 参数，若镜像所支持的架构不在 -a | --arch 参数所提供的架构列表内，
# 不输出 Mirror 失败的错误信息，并不将镜像名称保存至 Mirror 失败的列表内
# FYI: https://github.com/cnrancher/hangar/issues/24
# 默认为 true
hangar mirror -f ./list.txt -a arm64 --no-arch-failed=false

# 使用 -j, --jobs 参数，指定 Worker 数量，并发拷贝镜像（支持 1~20 个 jobs）
hangar mirror -f ./list.txt -j 10    # 启动 10 个 Worker

# 使用 --repo-type 指定目标镜像仓库的类型，默认为空字符串，可设定为 "harbor"
# 目标镜像仓库的类型为 harbor 时，将会自动为目标镜像创建 project
hangar mirror -f ./list.txt --repo-type=harbor

# 使用 --default-project 参数指定默认的 project 名称
# 默认值为 library
# 此参数会将 `private.io/mysql:5.8` 这种镜像重命名为 `private.io/library/mysql:5.8`
hangar mirror -f ./list.txt --default-project=library

# 使用 -o, --failed 参数，将 mirror 失败的镜像列表输出至指定文件中
# 默认输出至 mirror-failed.txt
hangar mirror -f image-list.txt -o failed-list.txt

# 使用 --tls-verify=false 参数，跳过 Registry 仓库的 TLS 验证
hangar mirror -f ./list.txt --tls-verify=false

# 使用 --debug 参数，输出更详细的调试日志
hangar mirror --debug
```
