# Save

```console
$ ./image-tools save -h
Usage of save:
  -a string
        architecture list of images, separate with ',' (default "amd64,arm64")
  -compress string
        compress format, can be 'gzip', 'zstd' or 'dir' (default "gzip")
  -d string
        Output saved images into destination file (directory or tar tarball) (default "saved-images.tar.gz")
  -debug
        enable the debug output
  -f string
        image list file
  -j int
        job number, async mode if larger than 1, maximum is 20 (default 1)
  -o string
        file name of the save failed image list (default "save-failed.txt")
  -part
        enable segment compress
  -part-size string
        segment part size (a number, or a string ended with 'K','M' or 'G') (default "2G")
  -s string
        override the source registry
```

## 准备工作

在执行 `image-tools save` 之前，若镜像列表中存在私有镜像，请确保手动执行过 `docker login <registry-url>` 登录。

## 镜像列表格式

每一行包含 **“镜像名称:TAG”**，镜像与 TAG 之间以 `:` 分隔，例如：

```txt
# <NAME>:<TAG>
rancher/rancher:v2.7.0
```

> 若该行以 `#` 或 `//` 开头，那么这一行将被视为注释

## QuickStart

将 `rancher-images.txt` 列表中的所有镜像下载到本地并创建 `tar.gz` 压缩包：

```sh
./image-tools save -f ./rancher-images.txt -d saved-images.tar.gz
```

> 此命令会先将镜像下载至 `saved-image-cache` 缓存文件夹内，之后对此文件夹创建压缩包。

## Parameters

使用样例 & 命令行参数：

```sh
# 使用 -f (file) 参数指定镜像列表文件
./image-tools save -f ./list.txt

# 使用 -d (destination) 参数，指定导出镜像的文件名称
# 可配合 -compress 参数使用
# 默认文件名为 saved-images.tar.gz
./image-tools save -f ./list.txt -d saved-images.tar.gz

# 使用 -s (source) 参数，可以在不修改镜像列表的情况下，指定源镜像的 registry
# 如果镜像列表中的源镜像没有写 registry，且未设定 -s 参数，那么源镜像的 registry 会被设定为默认的 docker.io
./image-tools save -f ./list.txt -s custom.registry.io -d saved-images.tar.gz

# 使用 -a (arch) 参数，指定导出的镜像的架构（以逗号分隔）
# 默认为 amd64,arm64
./image-tools save -f ./list.txt -a amd64,arm64 -d saved-images.tar.gz

# 使用 -j (jobs) 参数，指定协程池数量，并发下载镜像至本地（支持 1~20 个 jobs）
./image-tools save -f ./list.txt -d saved-images.tar.gz -j 10 # 启动 10 个 Worker

# 使用 -part 参数，启用分卷压缩，默认每个卷的大小为 2G
# 可使用 -part-size 参数设定卷大小
# 开启分卷压缩后，将创建以 .part* 为后缀的文件
./image-tools save -f ./list.txt -d saved-images.tar.gz -part -part-size=4G # 指定每个卷大小为 4G

# 在不设定 -f 参数时，可手动按行输入镜像列表，下载某一个镜像
# 此时将不支持并发拷贝
# 注意在此模式下，使用 `Ctrl-D` 结束镜像列表的输入，不要使用 `Ctrl-C` 结束程序，否则将无法创建压缩包！
./image-tools save -d saved-images.tar.gz
......
>>> rancher/rancher:v2.7.0

# 使用 -o (output) 参数，将 save 失败的镜像列表输出至指定文件中
# 默认输出至 save-failed.txt
./image-tools save -f image-list.txt -o failed-list.txt

# 使用 -compress 参数，指定压缩格式
# 可选：gzip, zstd, dir
# 默认为 gzip 格式，若为 dir 格式则表示只将 save 的镜像保存在文件夹中，不对其进行压缩
./image-tools save -f image-list.txt -compress=zstd -d saved.tar.zstd

# 使用 -debug 参数，输出更详细的调试日志
./image-tools save -debug
```

## Save 原理

**本工具 Save 命令创建的压缩包与 `docker save` 创建的压缩包不通用。**

本工具在执行 Save 时，首先将镜像列表中的镜像的 blobs 使用 `skopeo copy` 以 OCI 格式保存至本地的 `saved-image-cache` 文件夹中。

镜像的 blobs 文件会保存至 `saved-image-cache/share` 共享文件夹内。

待所有镜像下载完毕后，本工具会对 `saved-image-cache` 创建压缩包（使用 `-compress=dir` 参数除外）。

创建完压缩包后，本工具不会自动删除 `saved-image-cache` 文件夹，请手动删除此文件夹以节省硬盘空间。

## 分卷压缩

可使用 `-part` 参数启用分卷压缩，并使用 `-part-size` 参数指定卷大小，该参数支持纯数字（表示字节大小）或以`K`、`M`、`G`字母结尾的数值。

启用分卷压缩后，创建的压缩包文件将以 `.part*` 后缀结尾。

本工具创建的分卷压缩的原理与 Linux 命令 `split` 一致，除了可使用本工具的 `load` 命令解压外，还可以使用以下命令对分卷压缩包进行解压：

```sh
# 将所有分卷整合
cat ./saved-images.tar.gz.part* > saved-images.tar.gz
# 对整合后的压缩包解压
tar -zxvf ./saved-images.tar.gz

# 或使用以下命令一键解压
cat ./saved-images.tar.gz.part* | tar -zxv
```

> 可使用镜像推送工具的 `load` 命令配合 `-compress=dir` 参数，从解压后的缓存文件夹中加载镜像上传至私有仓库。

## Logs

执行该工具输出的日志包含了 “时间、日志的等级”，在并发拷贝镜像时每行日志的 `M_ID`（对应镜像列表中的第 N 个 Manifest 列表）和 `IMG_ID`（该 Manifest 列表中的第 N 个镜像）可用来跟踪具体是哪个镜像下载失败。

## Output

此工具最终会生成一个 `tar.gz` 压缩包文件。

若拷贝过程中某个镜像拷贝失败，那么该工具会将拷贝失败的镜像列表输出至 `save-failed.txt`，可使用 `-o` 参数设定拷贝失败的镜像列表的文件名称。
