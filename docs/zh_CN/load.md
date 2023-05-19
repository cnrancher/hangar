# Load

Load 命令将 Save 命令保存的镜像文件导入至目标 Registry 仓库中，并为导入的镜像创建 Manifest 列表。

## QuickStart

将 `save` 指令保存的压缩文件导入至 `docker.io` 中：

```sh
hangar load -s ./saved-images.tar.gz -d docker.io
```

若待导入的文件格式不是 `tar.gz` 时，请使用 `--compress` 参数指定文件格式：

```sh
# 文件格式为解压后的 cache 文件夹
hangar load -s ./saved-image-cache -d docker.io --compress=dir

# 文件压缩格式为 zstd
hangar load -s ./saved-images.tar.zstd -d docker.io --compress=zstd
```

若待导入的文件为 Save 命令生成的分片压缩文件时，Load 命令会自动识别压缩文件名称的 `.partX`  后缀：

```console
$ ls -alh
drwxr-xr-x   6 root  root   192B  1  6 18:00 .
drwxr-x---+ 70 root  root   2.2K  1  6 18:00 ..
-rw-r--r--   1 root  root    50M  1  6 17:59 saved-images.tar.gz.part0
-rw-r--r--   1 root  root    50M  1  6 17:59 saved-images.tar.gz.part1
-rw-r--r--   1 root  root    50M  1  6 17:59 saved-images.tar.gz.part2
-rw-r--r--   1 root  root   5.3M  1  6 17:59 saved-images.tar.gz.part3

$ hangar load -s saved-images.tar.gz -d private.registry.io
18:01:28 [INFO] Decompressing saved-images.tar.gz...
18:01:28 [INFO] Read "saved-images.tar.gz.part0"
18:01:28 [INFO] Read "saved-images.tar.gz.part1"
18:01:28 [INFO] Read "saved-images.tar.gz.part2"
18:01:28 [INFO] Read "saved-images.tar.gz.part3"
......
```

### Harbor V2

若目标镜像仓库类型为 Harbor V2，那么可使用 `--repo-type=harbor` 参数，该参数会在导入时自动为 Harbor V2 仓库创建 Project。

> 若 Harbor V2 为 HTTP，还需要添加 `--harbor-https=false` 参数。

若 Save 时镜像列表中的目标镜像不包含 `Project` （例如 `mysql:8.0`, `busybox:latest`），那么在 Load 的过程中会自动为其添加 `library` Project 前缀（`library/mysql:8.0`，`library/busybox:latest`）。

可使用 `--default-project=library` 参数设定添加 Project 的名称 （默认为 `library`）。

## Parameters

命令行参数：

```sh
# 使用 -s, --source 参数指定导入的文件（必选参数）
hangar load -s ./saved-images.tar.gz

# 使用 -d, --destination 参数，指定目标镜像的 registry
# 优先级为：-d 参数 > DOCKER_REGISTRY 环境变量
hangar load -s ./saved-images.tar.gz -d private.registry.io

# 使用 --compress 参数，指定导入文件的压缩格式
# 可选：gzip, zstd, dir
# 默认为 gzip 格式，若为 dir 格式则表示从文件夹中加载镜像，不对其进行解压
hangar load -s ./saved-images.tar.zstd --compress=zstd

# 使用 --repo-type 指定目标镜像仓库的类型，默认为空字符串，可设定为 "harbor"
# 目标镜像仓库的类型为 harbor 时，将会自动为目标镜像创建 project
hangar load -s ./saved-images.tar.gz -d private.registry.io --repo-type=harbor

# 使用 --default-project 参数指定默认的 project 名称
# 默认值为 library
# 此参数会将 `docker.io/mysql:5.8` 这种镜像重命名为 `docker.io/library/mysql:5.8`
hangar load -s ./saved-images.tar.gz -d private.registry.io --default-project=library

# 使用 -j, --jobs 参数，指定协程池数量，并发导入镜像（支持 1~20 个 jobs）
hangar load -s ./saved-images.tar.gz -j 10    # 启动 10 个协程

# 使用 -o, --output 参数，将 load 失败的镜像列表输出至指定文件中
# 默认输出至 mirror-failed.txt
hangar load -s ./saved-images.tar.gz -o failed-list.txt

# 使用 --tls-verify=false 参数，跳过 Registry 仓库的 TLS 验证
hangar load -s ./saved-images.tar.gz --tls-verify=false

# 使用 --debug 参数，输出更详细的调试日志
hangar load -s ./saved-images.tar.gz --debug
```

## 加载分卷压缩包

Load 子命令支持加载 Save 生成的分卷 (part) 压缩包，文件名应当以 `.partX` 为后缀，以 `.part0`、`.part1`、`.part2`……顺序排列。在加载分卷压缩包时，`-s` 参数指定的源文件名应当不包含 `.partX` 后缀（例如 `saved-images.tar.gz`），该工具会自动识别分卷压缩包并按顺序从 `part0`、`part1`……读取数据解压。

```console
$ ls -alh
drwxr-xr-x   6 root  root   192B  1  6 18:00 .
drwxr-x---+ 70 root  root   2.2K  1  6 18:00 ..
-rw-r--r--   1 root  root    50M  1  6 17:59 saved-images.tar.gz.part0
-rw-r--r--   1 root  root    50M  1  6 17:59 saved-images.tar.gz.part1
-rw-r--r--   1 root  root    50M  1  6 17:59 saved-images.tar.gz.part2
-rw-r--r--   1 root  root   5.3M  1  6 17:59 saved-images.tar.gz.part3

$ hangar load -s saved-images.tar.gz -d private.registry.io
18:01:28 [INFO] Decompressing saved-images.tar.gz...
18:01:28 [INFO] Read "saved-images.tar.gz.part0"
18:01:28 [INFO] Read "saved-images.tar.gz.part1"
18:01:28 [INFO] Read "saved-images.tar.gz.part2"
18:01:28 [INFO] Read "saved-images.tar.gz.part3"
......
```

> 请不要尝试单独 Load 某一个 Part 文件，因文件不完整，无法解压。

## 加载不同架构的镜像包

> `v1.6.0` 及后续版本支持此功能

Load 命令支持导入不同架构的镜像包，请参照下面的例子了解此特性的用法：

使用 Hangar Save 命令创建了多个不同架构的压缩包，例如：

```sh
# 样例镜像列表
cat list.txt
docker.io/library/nginx:1.22
docker.io/library/nginx:1.23

# 仅生成包含 AMD64 架构的镜像包
hangar save -f list.txt -a "amd64" -d amd64-images.tar.gz

# 仅生成包含 ARM64 架构的镜像包
hangar save -f list.txt -a "arm64" -d arm64-images.tar.gz
```

Hangar 的 Load 命令支持依次导入此例子中的 `amd64-images.tar.gz` 和 `arm64-images.tar.gz` 至 Registry Server 中，最终构建的 Manifest List 包含两种架构的镜像索引。

```sh
# 先导入仅包含 AMD64 架构的镜像包至 Registry Server
hangar load -s amd64-images.tar.gz -d <REGISTRY_URL>

# 此时查看已导入的镜像的 Manifest List，仅包含 ARM64 架构
skopeo inspect docker://<REGISTRY_URL>/library/nginx:1.22 --raw | jq
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.docker.distribution.manifest.list.v2+json",
  "manifests": [
    {
      "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
      "size": 1235,
      "digest": "sha256:66f1a9ae96f5a18068fcbd53e0171c78b40adffa3d70f565341eb453a34bb099",
      "platform": {
        "architecture": "arm64",
        "os": "linux",
        "variant": "v8"
      }
    }
  ]
}

# 再导入包含 ARM64 架构的镜像包至 Registry Server
hangar load -s arm64-images.tar.gz -d <REGISTRY_URL>

# 导入两种架构的镜像包后，查看导入后的镜像的 Manifest List，包含 AMD64 和 ARM64 两种架构
skopeo inspect docker://<REGISTRY_URL>/library/nginx:1.22 --raw | jq
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.docker.distribution.manifest.list.v2+json",
  "manifests": [
    {
      "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
      "size": 1235,
      "digest": "sha256:66f1a9ae96f5a18068fcbd53e0171c78b40adffa3d70f565341eb453a34bb099",
      "platform": {
        "architecture": "arm64",
        "os": "linux",
        "variant": "v8"
      }
    },
    {
      "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
      "size": 1235,
      "digest": "sha256:7dcde3f4d7eec9ccd22f2f6873a1f0b10be189405dcbfbaac417487e4fb44c4b",
      "platform": {
        "architecture": "amd64",
        "os": "linux"
      }
    }
  ]
}
```
