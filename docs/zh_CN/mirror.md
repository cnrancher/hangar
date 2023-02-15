# Mirror

```console
$ ./hangar mirror -h
Usage of mirror:
  -a string
        architecture list of images, separate with ',' (default "amd64,arm64")
  -d string
        override the destination registry
  -debug
        enable the debug output
  -default-project string
        project name when dest repo type is harbor and dest project is empty (default "library")
  -f string
        image list file
  -harbor-https
        use HTTPS by default when create harbor project (default true)
  -j int
        job number, async mode if larger than 1, maximun is 20 (default 1)
  -o string
        file name of the mirror failed image list (default "mirror-failed.txt")
  -repo-type string
        repository type, can be 'harbor' or empty
  -s string
        override the source registry
```
## 镜像列表格式

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
./hangar mirror -f ./image-list.txt -d <dest-registry-url>
```

### Harbor V2

若目标镜像仓库类型为 Harbor V2，那么可使用 `-repo-type=harbor` 参数，该参数会自动为 Harbor V2 仓库创建 Project。

除此之外若镜像列表中的目标镜像不包含 `Project` （例如 Docker Hub 的 `mysql:8.0`, `busybox:latest`），那么在 mirror 的过程中会自动为其添加 `library` Project 前缀（`library/mysql:8.0`，`library/busybox:latest`）。

可使用 `-default-project=library` 参数设定添加 Project 的名称 （默认为 `library`）。

## Parameters

命令行参数：

```sh
# 使用 -f (file) 参数指定镜像列表文件
./hangar mirror -f ./list.txt

# 使用 -d (destination) 参数，可以在不修改镜像列表的情况下，指定目标镜像的 registry
# 如果列表中的目标镜像没有写 registry，且未设定 -d 参数和 DOCKER_REGIRTSY 环境变量，那么目标镜像的 registry 会被设定为默认的 docker.io
# 优先级为：-d 参数 > DOCKER_REGISTRY 环境变量 > 镜像列表中已写好的 registry
./hangar mirror -f ./list.txt -d private.registry.io

# 使用 -s (source) 参数，可以在不修改镜像列表的情况下，指定源镜像的 registry
# 如果镜像列表中的源镜像没有写 registry，且未设定 -s 参数，那么源镜像的 registry 会被设定为默认的 docker.io
./hangar mirror -f ./list.txt -s docker.io

# 使用 -a (arch) 参数，设定拷贝镜像的架构（以逗号分隔）
# 默认为 amd64,arm64
./hangar mirror -f ./list.txt -a amd64,arm64

# 使用 -j (jobs) 参数，指定协程池数量，并发拷贝镜像（支持 1~20 个 jobs）
./hangar mirror -f ./list.txt -j 10    # 启动 10 个 Worker

# 在不设定 -f 参数时，可手动按行输入镜像列表，拷贝某一个镜像
# 此时将不支持并发拷贝
./hangar mirror
......
>>> hello-world library/hello-world latest

# 使用 -repo-type 指定目标镜像仓库的类型，默认为空字符串，可设定为 "harbor"
# 目标镜像仓库的类型为 harbor 时，将会自动为目标镜像创建 project
./hangar mirror -f ./list.txt -repo-type=harbor

# 使用 -default-project 参数指定默认的 project 名称
# 默认值为 library
# 此参数会将 `private.io/mysql:5.8` 这种镜像重命名为 `private.io/library/mysql:5.8`
./hangar mirror -f ./list.txt -repo-type=harbor -default-project=library

# 使用 -o (output) 参数，将 mirror 失败的镜像列表输出至指定文件中
# 默认输出至 mirror-failed.txt
./hangar mirror -f image-list.txt -o failed-list.txt

# 使用 -debug 参数，输出更详细的调试日志
./hangar mirror -debug
```

## Logs

执行该工具输出的日志包含了 “时间、日志的等级”，在并发拷贝镜像时每行日志的 `M_ID`（对应镜像列表中的第 N 个 Manifest 列表）和 `IMG_ID`（该 Manifest 列表中的第 N 个镜像）可用来跟踪具体是哪个镜像拷贝失败。

## Output

若拷贝过程中某个镜像拷贝失败，那么该工具会将拷贝失败的镜像列表输出至 `mirror-failed.txt`，可使用 `-o` 参数设定拷贝失败的镜像列表的文件名称。
