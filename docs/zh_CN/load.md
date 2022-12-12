# Load

```console
$ image-tools load -h
Usage of load:
  -d string
        override the destination registry
  -debug
        enable the debug output
  -j int
        job number, async mode if larger than 1, maximum is 20 (default 1)
  -o string
        file name of the load failed image list (default "load-failed.txt")
  -s string
        saved tar.gz file
```

## QuickStart

将 `save` 指令导出的 `tar.gz` 压缩包导入至 `private.registry.io` 中：

```sh
./image-tools load -s ./saved-images.tar.gz -d private.registry.io
```

此命令会将自动根据 `save` 时保存的镜像文件在目标 registry 中创建适配多架构的 Manifest 列表。

## Parameters

命令行参数：

```sh
# 使用 -s (source file) 参数指定导入的 tar.gz 文件（必选参数）
./image-tools load -s ./saved-images.tar.gz

# 使用 -d (destination) 参数，指定目标镜像的 registry
# 优先级为：-d 参数 > DOCKER_REGISTRY 环境变量
./image-tools load -s ./saved-images.tar.gz -d private.registry.io

# 使用 -j (jobs) 参数，指定协程池数量，并发导入镜像（支持 1~20 个 jobs）
./image-tools load -s ./saved-images.tar.gz -j 10    # 启动 10 个协程

# 使用 -o (output) 参数，将 load 失败的镜像列表输出至指定文件中
# 默认输出至 mirror-failed.txt
./image-tools load -s ./saved-images.tar.gz -o failed-list.txt

# 使用 -debug 参数，输出更详细的调试日志
./image-tools load -s ./saved-images.tar.gz -debug
```

## Logs

执行该工具输出的日志包含了 “时间、日志的等级”，在并发拷贝镜像时每行日志还包含了 `M_ID`（对应导入的第 N 个 Manifest 列表）和 `IMG_ID`（该 Manifest 列表中的第 N 个镜像），在并发拷贝遇到错误时可根据这两个 ID 来跟踪具体是哪个镜像拷贝失败。

## Output

若拷贝过程中某个镜像导入失败，那么该工具会将拷贝失败的镜像列表输出至 `load-failed.txt`，可使用 `-o` 参数设定导入失败的镜像列表的文件名称。
