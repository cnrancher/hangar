# load-validate

```console
$ image-tools load-validate -h
Usage of load-validate:
  -compress string
        compress format, can be 'gzip', 'zstd' or 'dir' (default "gzip")
  -d string
        target private registry:port
  -debug
        enable the debug output
  -default-project string
        project name when project is empty (default "library")
  -j int
        job number, async mode if larger than 1, maximun is 20 (default 1)
  -o string
        file name of the validate failed image list (default "load-validate-failed.txt")
  -s string
        saved file to load (tar tarball or a directory)
```

## QuickStart

在执行 `load` 命令后，对已 Load 过的镜像进行验证，确保镜像已经被 Load 到目标仓库，验证失败的镜像列表会保存在 `load-validate-failed.txt` 文件中。

输入的文件为 Save 子命令保存的压缩包文件或解压后的文件夹目录名。

```sh
./image-tools load-validate -s ./saved-images.tar.gz -d private.registry.io
```

## Parameters

命令行参数：

```sh
# 使用 -s (source) 参数，设定 Save 保存的文件名称
# 使用 -d (destination) 参数，设定目标镜像 registry
./image-tools load-validate -s ./saved-images.tar.gz -d private.registry.io

# 使用 -j (jobs) 参数，设定协程池数量，并发校验镜像（支持 1~20 个 jobs）
./image-tools load-validate -s ./saved-images.tar.gz -d private.registry.io -j 10 # 启动 10 个 Worker

# 使用 -compress 参数，指定导入的文件的压缩类型
# 可选：gzip, zstd, dir
# 默认为 gzip 格式，若为 dir 格式则表示从文件夹中加载镜像进行校验，不对其解压
./image-tools load-validate -s ./saved-image-cache -d private.registry.io -compress=dir

# 使用 -default-project 参数指定默认的 project 名称
# 默认值为 library
# 此参数会将 `private.io/mysql:5.8` 这种镜像重命名为 `private.io/library/mysql:5.8`
./image-tools load-validate -s ./saved-image-cache -d private.registry.io -default-project=library

# 使用 -o (output) 参数，将校验失败的镜像列表输出至指定文件中
# 默认输出至 load-validate-failed.txt
./image-tools load-validate -s ./saved-images.tar.gz -d private.registry.io -o failed.txt

# 使用 -debug 参数，输出更详细的调试日志
./image-tools load-validate -s ./saved-images.tar.gz -d private.registry.io -debug
```

# FAQ

使用校验功能时可能遇到的报错及原因：

1. 报错：`Validate failed: destination manifest MIME type unknow: application/vnd.docker.distribution.manifest.v2+json`

    在目标镜像的 Manifest 的 MediaType 不是 `"application/vnd.docker.distribution.manifest.list.v2+json"` 时会出现此报错。

    可使用 `skopeo inspect docker://<dest-image>:<tag> --raw` 检查目标镜像的 Manifest 的 MediaType 种类。

1. 报错： `destination manifest does not exists`，表示目标镜像不存在，请检查目标镜像。

1. 遇到下面报错：

    ```text
    11:22:33 [ERRO] [M_ID:1] srcSpec: [
        {
            "digest": "",
            "platform": {
                "architecture": "amd64",
                "os": "linux"
            }
        }
    ]
    11:22:33 [ERRO] [M_ID:1] dstSpec: [
        {
            "digest": "",
            "platform": {
                "architecture": "amd64",
                "os": "windows"
                "os.version": "1.0.10"
            }
        }
    ]
    ```

    表示本地的镜像 (srcSpec) 与服务器中的镜像 (dstSpec) 的某些字段不符合

# Logs

执行该工具输出的日志包含了 “时间、日志的等级”，在并发校验镜像时每行日志的 `M_ID`，可用来跟踪具体是哪个镜像校验失败。

## Output

若校验过程中某个镜像校验失败，那么该工具会将校验失败的镜像列表输出至 `load-validate-failed.txt`，可使用 `-o` 参数设定校验失败的镜像列表的文件名称。
