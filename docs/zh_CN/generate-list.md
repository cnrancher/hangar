# generate-list

```console
$ image-tools generate-list -h
Usage of generate-list:
  -chart value
        chart path
  -debug
        enable the debug output
  -dev
        Switch to dev branch/url of charts & KDM data
  -git-token string
        git token for download image data from RPM GC private repos
  -kdm string
        kdm path/url
  -kubeversion string
        minimum kuber version (semantic version with 'v' prefix) (default "v1.21.0")
  -o string
        generated image list path (linux and windows images) (default "generated-list.txt")
  -output-linux string
        generated linux image list
  -output-source string
        generate image list with image source
  -output-windows string
        generated windows image list
  -rancher string
        rancher version (semver with 'v' prefix)(use '-ent' suffix to distinguish GC)
  -registry string
        override the registry url
  -system-chart value
        system chart path
```

## QuickStart

根据 Rancher 版本号，获取最新的 KDM 数据，并自动克隆 Chart 仓库到本地，生成镜像列表：

```sh
./image-tools generate-list -rancher="v2.7.0-ent"
```

## Parameters

命令行参数：

```sh
# 使用 -rancher 参数，指定 Rancher 版本号，以 `-ent` 结尾为 RPM GC 版本
# 若只指定 Rancher 版本号，该工具将自动根据 Rancher 版本号下载对应的 KDM 数据，
# 并克隆 charts 仓库到本地，从中生成镜像列表文件
./image-tools generate-list -rancher="v2.7.0"

# 使用 -registry 参数，指定生成镜像的 Registry 地址（默认为空字符串）
./image-tools generate-list -rancher="v2.7.0" -registry="docker.io"

# 使用 -o 参数，指定输出的镜像列表文件名称（默认为 generated-list.txt）
./image-tools generate-list -rancher="v2.7.0" -o ./generated-list.txt

# 使用 -output-linux 参数，指定输出的 Linux 镜像列表文件名称
# 默认情况下本工具不会单独生成 Linux 镜像列表文件
./image-tools generate-list -rancher="v2.7.0" -output-linux ./generated-list-linux.txt

# 使用 -output-source 参数，指定输出的包含镜像来源的列表文件名称
# 默认情况下本工具不会生成包含镜像来源的列表文件
./image-tools generate-list -rancher="v2.7.0" -output-source ./generated-list-source.txt

# 使用 -output-windows 参数，指定输出的 Windows 镜像列表文件名称
# 默认情况下本工具不会单独生成 Windows 镜像列表文件
./image-tools generate-list -rancher="v2.7.0" -output-windows ./generated-list-windows.txt

# 使用 -dev 参数，在没有使用 -chart, -system-chart, -kdm 参数时，
# 自动从 KDM 和 chart 的 dev 分支生成镜像列表
# 默认情况下此工具会从 release 分支生成镜像列表
./image-tools generate-list -rancher="v2.7.0" -dev

# 使用 -kubeversion 参数，指定最小的 Kubernetes 版本（默认 v1.21.0）
./image-tools generate-list -rancher="v2.7.0" -kubeversion="v1.21.0"

# 使用 -chart 参数，指定 chart 仓库的路径
./image-tools generate-list -rancher="v2.7.0" -chart ./charts

# 使用 -system-chart 参数，指定 system-chart 仓库的路径
./image-tools generate-list -rancher="v2.7.0" -system-chart ./system-chart

# 使用 -kdm 参数，指定 KDM Data 文件的位置或 URL 链接
./image-tools generate-list -rancher="v2.7.0" -kdm ./data.json
./image-tools generate-list -rancher="v2.7.0" -kdm https://releases.rancher.com/kontainer-driver-metadata/release-v2.7/data.json

# 使用 -debug 参数，输出更详细的调试日志
./image-tools generate-list -rancher="v2.7.0" -debug
```

### 自定义 KDM 文件和 Chart 仓库

执行此工具时，如果只指定 `-rancher` 命令行参数，将自动根据 Rancher 版本获取 KDM 数据并克隆 Chart 仓库到本地。除此之外可通过 `-chart`、`-system-chart`、`-kdm` 参数自定义生成镜像列表时读取的 KDM 数据文件和 Chart 仓库。

> 在有多个 chart 和 system-chart 仓库需要加载时，可指定多个 `-chart` 和 `-system-chart` 参数。

```sh
# 首先下载 KDM data.json，克隆 chart 仓库到本地
./image-tools generate-list \
    -rancher="v2.7.0" \
    -kubeversion="v1.21.0" \
    -kdm ./data.json \
    -chart ./charts-1 \
    -chart ./charts-2 \
    -system-chart ./system-charts-1 \
    -system-chart ./system-charts-2
```

## Output

此工具最终会从 Chart 仓库和 KDM 数据中生成一份包含 Windows 和 Linux 镜像的列表文件，若需要查看镜像的来源，可通过添加 `-output-source="FILE_NAME.txt"` 参数额外输出一份包含镜像来源的列表文件。
