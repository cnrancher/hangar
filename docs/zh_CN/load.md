# Load

```console
$ image-tools load -h
Usage of load:
  -compress string
        compress format, can be 'gzip', 'zstd' or 'dir' (default "gzip")
  -d string
        target private registry:port
  -debug
        enable the debug output
  -default-project string
        project name when dest repo type is harbor and dest project is empty (default "library")
  -harbor-https
        use HTTPS by default when create harbor project (default true)
  -j int
        job number, async mode if larger than 1, maximum is 20 (default 1)
  -o string
        file name of the load failed image list (default "load-failed.txt")
  -repo-type string
        repository type, can be 'harbor' or empty
  -s string
        saved file to load (tar tarball or a directory)
```

## QuickStart

将 `save` 指令导出的 `tar.gz` 压缩包导入至 `private.registry.io` 中：

```sh
./image-tools load -s ./saved-images.tar.gz -d private.registry.io
```

此命令会将自动根据 `save` 时保存的镜像文件在目标 registry 中创建适配多架构的 Manifest 列表。

### Harbor V2

若目标镜像仓库类型为 Harbor V2，那么可使用 `-repo-type=harbor` 参数，该参数会在导入时自动为 Harbor V2 仓库创建 Project。

除此之外若 Save 时镜像列表中的目标镜像不包含 `Project` （例如 Docker Hub 的 `mysql:8.0`, `busybox:latest`），那么在 Load 的过程中会自动为其添加 `library` Project 前缀（`library/mysql:8.0`，`library/busybox:latest`）。

可使用 `-default-project=library` 参数设定添加 Project 的名称 （默认为 `library`）。

## Parameters

命令行参数：

```sh
# 使用 -s (source file) 参数指定导入的文件（必选参数）
./image-tools load -s ./saved-images.tar.gz

# 使用 -d (destination) 参数，指定目标镜像的 registry
# 优先级为：-d 参数 > DOCKER_REGISTRY 环境变量
./image-tools load -s ./saved-images.tar.gz -d private.registry.io

# 使用 -compress 参数，指定导入文件的压缩格式
# 可选：gzip, zstd, dir
# 默认为 gzip 格式，若为 dir 格式则表示从文件夹中加载镜像，不对其进行解压
./image-tools load -s ./saved-images.tar.zstd -compress=zstd

# 使用 -repo-type 指定目标镜像仓库的类型，默认为空字符串，可设定为 "harbor"
# 目标镜像仓库的类型为 harbor 时，将会自动为目标镜像创建 project
./image-tools load -s ./saved-images.tar.gz -d private.registry.io -repo-type=harbor

# 使用 -default-project 参数指定默认的 project 名称
# 默认值为 library
# 此参数会将 `private.io/mysql:5.8` 这种镜像重命名为 `private.io/library/mysql:5.8`
./image-tools load -s ./saved-images.tar.gz -d private.registry.io -repo-type=harbor -default-project=library

# 使用 -j (jobs) 参数，指定协程池数量，并发导入镜像（支持 1~20 个 jobs）
./image-tools load -s ./saved-images.tar.gz -j 10    # 启动 10 个协程

# 使用 -o (output) 参数，将 load 失败的镜像列表输出至指定文件中
# 默认输出至 mirror-failed.txt
./image-tools load -s ./saved-images.tar.gz -o failed-list.txt

# 使用 -debug 参数，输出更详细的调试日志
./image-tools load -s ./saved-images.tar.gz -debug
```

## 加载分卷压缩包

Load 子命令支持加载 Save 生成的分卷 (part) 压缩包，文件名应当以 `.part*` 为后缀，以 `.part0`、`.part1`、`.part2`……顺序排列。在加载分卷压缩包时，`-s` 参数指定的源文件名应当不包含 `.part*` 后缀（例如 `saved-images.tar.gz`），该工具会自动识别分卷压缩包并按顺序从 `part0`、`part1`……读取数据解压。

```console
$ ls -alh
drwxr-xr-x   6 root  root   192B  1  6 18:00 .
drwxr-x---+ 70 root  root   2.2K  1  6 18:00 ..
-rw-r--r--   1 root  root    50M  1  6 17:59 saved-images.tar.gz.part0
-rw-r--r--   1 root  root    50M  1  6 17:59 saved-images.tar.gz.part1
-rw-r--r--   1 root  root    50M  1  6 17:59 saved-images.tar.gz.part2
-rw-r--r--   1 root  root   5.3M  1  6 17:59 saved-images.tar.gz.part3

$ image-tools load -s saved-images.tar.gz -d private.registry.io
18:01:28 [INFO] Decompressing saved-images.tar.gz...
18:01:28 [INFO] Read "saved-images.tar.gz.part0"
18:01:28 [INFO] Read "saved-images.tar.gz.part1"
18:01:28 [INFO] Read "saved-images.tar.gz.part2"
18:01:28 [INFO] Read "saved-images.tar.gz.part3"
......
```

> 请不要尝试单独 Load 某一个 Part 文件，会遇到解压失败的错误，并导致解压出来的文件是受损的。

## Logs

执行该工具输出的日志包含了 “时间、日志的等级”，在并发拷贝镜像时每行日志的 `M_ID`（对应导入的第 N 个 Manifest 列表）和 `IMG_ID`（该 Manifest 列表中的第 N 个镜像）可用来跟踪具体是哪个镜像拷贝失败。

## Output

若拷贝过程中某个镜像导入失败，那么该工具会将拷贝失败的镜像列表输出至 `load-failed.txt`，可使用 `-o` 参数设定导入失败的镜像列表的文件名称。
