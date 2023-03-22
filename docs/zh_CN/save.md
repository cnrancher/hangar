# Save

Save 命令将镜像列表中的镜像保存至本地文件中，供 [Load](./load.md) 命令使用。

## 镜像列表格式

Save 命令的镜像列表格式与 `rancher-images.txt` 格式一致。

每一行包含 **“镜像名称:TAG”**，镜像与 TAG 之间以 `:` 分隔，例如：

```txt
# <NAME>:<TAG>
rancher/rancher:v2.7.0
```

> 若该行以 `#` 或 `//` 开头，那么这一行将被视为注释

## QuickStart

将 `rancher-images.txt` 列表中的所有镜像下载到本地并创建 `tar.gz` 压缩包：

```sh
hangar save -f ./rancher-images.txt -d saved-images.tar.gz
```

> 此命令会先将镜像下载至 `saved-image-cache` 缓存文件夹内，之后对此文件夹创建压缩包。

使用 `--compress` 参数，指定保存到本地的文件的压缩格式（或是否进行压缩）。

```sh
# 压缩格式为 zstd
hangar save -f ./rancher-images.txt -d saved-images.tar.zstd --compress=zstd

# 不进行压缩
hangar save -f ./rancher-images.txt -d saved-images --compress=dir
```

## Parameters

命令行参数：

```sh
# 使用 -f, --file 参数指定镜像列表文件
hangar save -f ./list.txt

# 使用 -d, --destination 参数，指定保存的镜像的文件名称
# (可配合 --compress 参数指定目标文件的格式)
# 默认文件名为 saved-images.tar.gz
hangar save -f ./list.txt -d saved-images.tar.gz

# 使用 -s, --source 参数，可在不修改镜像列表的情况下，指定源镜像的 registry
# 如果镜像列表中的源镜像没有写 registry，且未设定 -s 参数，那么源镜像的 registry 会被设定为默认的 docker.io
hangar save -f ./list.txt -s custom.registry.io -d saved-images.tar.gz

# 使用 -a, --arch 参数，指定导出的镜像的架构（以逗号分隔）
# 默认为 amd64,arm64
hangar save -f ./list.txt -a amd64,arm64 -d saved-images.tar.gz

# 使用 -j, --jobs 参数，指定 Worker 数量，并发下载镜像至本地（支持 1~20 个 jobs）
hangar save -f ./list.txt -d saved-images.tar.gz -j 10 # 启动 10 个 Worker

# 使用 --part 参数，启用分卷压缩，默认每个卷的大小为 2G
# 可使用 --part-size 参数设定卷大小
# 开启分卷压缩后，将创建以 .partX 为后缀的文件
hangar save -f ./list.txt -d saved-images.tar.gz --part --part-size=4G # 指定每个分片体积最大为 4G

# 使用 -o, --output 参数，将 save 失败的镜像列表输出至指定文件中
# 默认输出至 save-failed.txt
hangar save -f image-list.txt -o failed-list.txt

# 使用 --compress 参数，指定压缩格式/是否进行压缩
# 可选：gzip, zstd, dir
# 默认为 gzip 格式，若为 dir 格式则表示只将 save 的镜像保存在文件夹中，不对其进行压缩（重命名文件夹）
hangar save -f image-list.txt --compress=zstd -d saved.tar.zstd
hangar save -f image-list.txt --compress=dir -d saved-directory

# 使用 --debug 参数，输出更详细的调试日志
hangar save -f ./list.txt --debug
```

## Save 原理

> **Hangar Save 命令创建的压缩包与 `docker save` 创建的压缩包不通用。**

Hangar 在执行 Save 时，首先将镜像列表中的镜像的 `blobs` 使用 `skopeo copy` 以 OCI 格式保存至本地的 `saved-image-cache` 文件夹中。

镜像的 `blobs` 文件会保存至 `saved-image-cache/share` 共享文件夹内。

待所有镜像下载完毕后，Hangar 会对 `saved-image-cache` 创建压缩包（使用 `--compress=dir` 参数除外）。

创建完压缩包后，Hangar 不会自动删除 `saved-image-cache` 文件夹，请手动删除此文件夹以节省硬盘空间。

> 可使用 [sync](./sync.md) 以及 [compress](./compress.md) 命令对 cache 文件夹添加镜像/执行压缩操作。

## 分卷压缩

可使用 `--part` 参数启用分卷压缩，并使用 `--part-size` 参数指定卷大小，该参数支持纯数字（表示字节大小）或以`K`、`M`、`G`字母结尾的数值。

启用分卷压缩后，创建的压缩包文件将以 `.partX` 后缀结尾。

本工具创建的分卷压缩的原理与 Linux 命令 `split` 一致，除了可使用本工具的 `load` 命令解压外，还可以使用以下命令对分卷压缩包进行解压：

```sh
# 将所有分卷整合
cat ./saved-images.tar.gz.part* > saved-images.tar.gz
# 对整合后的压缩包解压
tar -zxvf ./saved-images.tar.gz

# 或使用以下命令一键解压
cat ./saved-images.tar.gz.part* | tar -zxv
```

> 可使用镜像推送工具的 `load` 命令配合 `--compress=dir` 参数，从解压后的缓存文件夹中加载镜像上传至私有仓库。
