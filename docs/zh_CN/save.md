# Save

```console
$ ./image-tools save -h
Usage of save:
  -a string
        architecture list of images, seperate with ',' (default "amd64,arm64")
  -d string
        Output saved images into tar.gz (default "saved-images.tar.gz")
  -debug
        enable the debug output
  -f string
        image list file
  -j int
        job number, async mode if larger than 1, maximum is 20 (default 1)
  -o string
        file name of the save failed image list (default "save-failed.txt")
  -s string
        override the source registry
```

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

# 使用 -d (destination) 参数，指定导出镜像的 tar.gz 文件名称
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

# 在不设定 -f 参数时，可手动按行输入镜像列表，拷贝某一个镜像
# 此时将不支持并发拷贝
# 注意在此模式下，使用 `Ctrl-D` 结束镜像列表的，不要使用 `Ctrl-C` 结束程序！
./image-tools save -d saved-images.tar.gz
......
>>> rancher/rancher:v2.7.0

# 使用 -o (output) 参数，将 save 失败的镜像列表输出至指定文件中
# 默认输出至 save-failed.txt
./image-tools save -f image-list.txt -o failed-list.txt

# 使用 -debug 参数，输出更详细的调试日志
./image-tools save -debug
```

## Logs

执行该工具输出的日志包含了 “时间、日志的等级”，在并发拷贝镜像时每行日志还包含了 `M_ID`（对应镜像列表中的第 N 个 Manifest 列表）和 `IMG_ID`（该 Manifest 列表中的第 N 个镜像），在并发下载遇到错误时可根据这两个 ID 来跟踪具体是哪个镜像下载失败。

## Output

若拷贝过程中某个镜像拷贝失败，那么该工具会将拷贝失败的镜像列表输出至 `mirror-failed.txt`，可使用 `-o` 参数设定拷贝失败的镜像列表的文件名称。
