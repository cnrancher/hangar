# mirror-validate

`mirror-validate` 命令用来校验已 Mirror 的镜像是否正确。

## QuickStart

在执行 `mirror` 命令后，对已 Mirror 过的镜像进行验证，确保镜像已经被 Mirror 到目标仓库，验证失败的镜像列表会保存在 `mirror-validate-failed.txt` 文件中。

**输入的镜像列表格式应当等同于 [Mirror](./mirror.md) 命令所支持的镜像列表格式。**

```sh
hangar mirror-validate -f ./list.txt -j 10
```

## Parameters

命令行参数：

```sh
# 使用 -f, --file 指定镜像列表文件
hangar mirror-validate -f ./list.txt

# 使用 -d, --destination 参数，设定目标镜像 registry
hangar mirror-validate -f ./list.txt -d private.registry.io

# 使用 -s, --source 参数，设定源镜像 registry
hangar mirror-validate -f ./list.txt -s docker.io

# 使用 -a, --arch 参数，设定镜像的架构（以逗号分隔）
# 默认为 amd64,arm64
hangar mirror-validate -f ./list.txt -a amd64,arm64,arm

# 使用 -j, --jobs 参数，设定协程池数量，并发校验镜像（支持 1~20 个 jobs）
hangar mirror-validate -f ./list.txt -j 20 # 启动 20 个 Worker

# 使用 -o, --output 参数，将校验失败的镜像列表输出至指定文件中
# 默认输出至 mirror-validate-failed.txt
hangar mirror-validate -f ./list.txt -o validate-failed-list.txt

# 使用 --debug 参数，输出更详细的调试日志
hangar mirror-validate -f ./list.txt --debug
```

# FAQ

使用校验功能时可能遇到的报错及原因：

1. 报错：`Validate failed: destination manifest MIME type unknow: application/vnd.docker.distribution.manifest.v2+json`

    在目标镜像的 Manifest 的 MediaType 不是 `"application/vnd.docker.distribution.manifest.list.v2+json"` 时会出现此报错。

    可使用 `skopeo inspect docker://<dest-image>:<tag> --raw` 检查目标镜像的 Manifest 的 MediaType 种类。

1. 报错： `destination manifest does not exists`，表示目标镜像不存在，请检查目标镜像。

1. 报错：`destination manifest list length should be 1`

    表示源镜像的 Manifest 只含有一个镜像，因此目标镜像的 Manifest List 列表中也应该只有一个镜像，若目标镜像的 Manifest List 列表有多个镜像时，会出现此报错。

    可使用 `skopeo inspect docker://<dest-image>:<tag> --raw` 查看目标镜像的 Manifest List 列表。

1. 报错：`source * != dest *` 表示源镜像与目标镜像的某些信息不匹配，例如 Arch、Variant、OS 等。

1. 遇到下面报错：

    ```text
    11:22:33 [ERRO] [M_ID:1] srcSpec: [
        {
            "digest": "sha256:9997c2f450f51e5c5402854899c42354b7968ca8298815df812b00409533527c",
            "platform": {
                "architecture": "amd64",
                "os": "linux"
            }
        }
    ]
    11:22:33 [ERRO] [M_ID:1] dstSpec: [
        {
            "digest": "sha256:8ace038ea3a18057e865b81e5ccd12d75ddeec0fdbd331555d877d39ac3f45bb",
            "platform": {
                "architecture": "amd64",
                "os": "linux"
            }
        }
    ]
    ```

    表示源镜像 (srcSpec) 的 Manifest List 与目标镜像 (dstSpec) 的 Manifest List 不符合，如果是 `digest` 不匹配，表示上游镜像已更新，私有仓库中的镜像还没有被更新，可重新运行 `mirror` 命令；若是其他字段不匹配 (`variant`, `os.version`) 等，也可通过重新运行 `mirror` 命令尝试修复。
