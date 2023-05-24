# load-validate

`load-validate` 命令在执行 `load` 后，对已导入的镜像进行校验，确保镜像已经被加载到目标 Registry，校验失败的镜像列表会保存在 `load-validate-failed.txt` 文件中。

## QuickStart

提供 Save 命令生成的文件，并指明目标 Registry URL。

```sh
hangar load-validate -s ./saved-images.tar.gz -d private.registry.io
```

除此之外，在使用 Load 命令加在容器镜像后，可使用 `--compress=dir` 和 `-s ./saved-image-cache` 指定输入的目录为 cache 目录，节省重复解压的时间。

```sh
hangar load-validate -s ./saved-image-cache -d private.registry.io --compress=dir
```

## Parameters

命令行参数：

```sh
# 使用 -s, --source 参数，设定 Save 保存的文件名称
# 使用 -d, --destination 参数，设定目标镜像 registry
hangar load-validate -s ./saved-images.tar.gz -d private.registry.io

# 使用 -j, --jobs 参数，设定 Worker 数量，并发校验镜像（支持 1~20 个 jobs）
hangar load-validate -s ./saved-images.tar.gz -d private.registry.io -j 10 # 启动 10 个 Worker

# 使用 --compress 参数，指定导入的文件的压缩类型
# 可选：gzip, zstd, dir
# 默认为 gzip 格式，若为 dir 格式则表示从文件夹中加载镜像进行校验，不对其解压
hangar load-validate -s ./saved-image-cache -d private.registry.io -compress=dir

# 使用 --default-project 参数指定默认的 project 名称
# 默认值为 library
# 此参数会将 `docker.io/mysql:5.8` 这种镜像重命名为 `docker.io/library/mysql:5.8`
hangar load-validate -s ./saved-image-cache -d private.registry.io --default-project=library

# 使用 -o, -output 参数，将校验失败的镜像列表输出至指定文件中
# 默认输出至 load-validate-failed.txt
hangar load-validate -s ./saved-images.tar.gz -d private.registry.io -o failed.txt

# 若 Registry Server 为 HTTP 或使用自签名 TLS Certificate，
# 需要使用 --tls-verify=false 参数，跳过 Registry 仓库的 TLS 验证
hangar load-validate -s ./saved-images.tar.gz --tls-verify=false

# 使用 --debug 参数，输出更详细的调试日志
hangar load-validate -s ./saved-images.tar.gz -d private.registry.io --debug
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
