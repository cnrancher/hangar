# Sync

`sync` 命令将额外的容器镜像保存在未压缩的 [Save](./save.md) 缓存文件夹中。

## 开发背景

Sync 命令以及 Compress、Decompress 命令为 Hangar 的高级特性，主要用于辅助 [Save](./save.md) 命令在保存镜像的 Blobs 时存在部分镜像下载失败的情况。

在 [Hangar Save](./save.md) 保存镜像的 Blobs 至缓存文件夹时，有些镜像可能因网络或其他原因下载失败，重新执行 [Hangar Save](./save.md) 命令会删除掉之前已有的缓存文件，并重新按照镜像列表下载镜像 Blobs 至缓存文件夹中，会浪费更多的时间，因此 Hangar 的 Sync 命令用于只将 Save 失败的镜像附加到 `saved-image-cache` 缓存文件夹中。

除此之外，Hangar 的 [Decompress](./decompress.md) 命令单独提供了解压 Hangar 创建的压缩包文件的功能，与 Load 命令的解压压缩包功能一致，支持 `gzip`, `zstd` 压缩格式和分片压缩功能。 Hangar 的 [Compress](./compress.md) 命令单独提供了压缩 Hangar 创建的缓存文件夹功能，与 Save 命令的创建压缩包的功能一致，支持将缓存文件夹创建为 `gzip`, `zstd` 格式的压缩包文件，且支持分片压缩功能。

将 Sync 和 Compress 命令结合使用的例子如下：

```sh
# 查看 Save 命令创建的缓存文件夹及压缩包文件
ls -al
-rw-r--r--@  1 user  staff    13B May  8 14:53 save-failed.txt
drwxr-xr-x@  6 user  staff   192B May  8 14:53 saved-image-cache
-rw-r--r--@  1 user  staff   107M May  8 14:53 saved-images.tar.gz

# 首先删掉 Save 命令创建的压缩文件
rm saved-images.tar.gz

# 使用 Sync 命令将 save-failed.txt 中的镜像下载至缓存文件夹 saved-image-cache
hangar sync -f ./save-failed.txt -d saved-image-cache -j 10

# 之后使用 Compress 命令创建压缩文件
hangar compress -f ./saved-image-cache
```

## QuickStart

使用 Sync 命令，将 `save-failed.txt` 中的镜像保存在 `saved-image-cache` 缓存目录中：

```sh
hangar sync -f ./save-failed.txt -d ./saved-image-cache -j 10
```

> Sync 失败的镜像会保存在 `sync-failed.txt`。

## Parameters

命令行参数：

```sh
# 使用 -f, --file 参数指定镜像列表文件
# 使用 -d, --destination 参数，指定同步镜像到目标文件夹目录
hangar sync -f ./list.txt -d [DIRECTORY]

# 使用 -s, --source 参数，可在不修改镜像列表的情况下，指定源镜像的 registry
# 如果镜像列表中的源镜像没有写 registry，且未设定 -s 参数，那么源镜像的 registry 会被设定为默认的 docker.io
hangar sync -f ./list.txt -s custom.registry.io -d [DIRECTORY]

# 使用 -a, --arch 参数，指定导出的镜像的架构（以逗号分隔）
# 默认为 amd64,arm64
hangar sync -f ./list.txt -d [DIRECTORY] -a amd64,arm64

# 使用 --os 参数，设定镜像的 OS（以逗号分隔）
# 默认为 linux,windows
hangar sync -f ./list.txt --os linux -d [DIRECTORY]

# 使用 --no-arch-os-fail 参数
# 若镜像所支持的架构不在 --arch 参数所提供的架构列表内，且镜像的 OS 不在 --os 参数所提供的系统列表内，
# 则将其视为镜像 Sync 失败，并输出错误日志。
# 默认为 false （仅输出 Warn 信息，不视为镜像 Sync 失败）
hangar sync -f ./list.txt -d [DIRECTORY] -a arm64 --no-arch-failed=false

# 使用 -j, --jobs 参数，指定 Worker 数量，并发下载镜像至本地（支持 1~20 个 jobs）
hangar sync -f ./list.txt -d [DIRECTORY] -j 10 # 启动 10 个 Worker

# 若 Registry Server 为 HTTP 或使用自签名 TLS Certificate，
# 需要使用 --tls-verify=false 参数，跳过 Registry 仓库的 TLS 验证
hangar sync -f ./list.txt -d [DIRECTORY] --tls-verify=false

# 使用 --debug 参数，输出更详细的调试日志
hangar sync -f ./list.txt -d [DIRECTORY] --debug
```

## Others

在使用 Sync 将镜像补充至缓存文件夹后，可使用 [compress](./compress.md) 命令压缩缓存文件夹，生成压缩包。
