# generate-list

## QuickStart

根据 Rancher 版本号，获取最新的 KDM 数据，并自动克隆 Chart 仓库到本地，生成镜像列表：

```sh
hangar generate-list --rancher="v2.7.0-ent"
```

> 以 `-ent` 结尾的 Rancher 版本号表示 RPM GC 版本。

此工具生成的镜像列表仅包含 KDM 和 Chart 仓库中与 Rancher 版本相匹配的镜像。因本工具筛选镜像的逻辑与
Rancher 生成的 `rancher-images.txt` 有差异，会与构建物中下载的镜像列表存在出入。

**此工具生成镜像列表时需要访问 GitHub 仓库等资源，请在良好的网络环境中使用此工具。**

## Parameters

命令行参数：

```sh
# 使用 --rancher 参数，指定 Rancher 版本号，以 `-ent` 结尾为 RPM GC 版本
# 若只指定 Rancher 版本号，该工具将自动根据 Rancher 版本号下载对应的 KDM 数据，
# 并克隆 charts 仓库到本地，从中生成镜像列表文件
hangar generate-list --rancher="v2.7.0"

# 使用 --registry 参数，指定生成镜像的 Registry 地址（默认为空字符串）
hangar generate-list --rancher="v2.7.0" --registry="docker.io"

# 使用 -o, --output 参数，指定输出的镜像列表文件名称（默认为 generated-list.txt）
hangar generate-list --rancher="v2.7.0" -o ./generated-list.txt

# 使用 --output-linux 参数，指定输出的 Linux 镜像列表文件名称
# 默认情况下本工具不会单独生成 Linux 镜像列表文件
hangar generate-list --rancher="v2.7.0" --output-linux ./generated-list-linux.txt

# 使用 --output-source 参数，指定输出的包含镜像来源的列表文件名称
# 默认情况下本工具不会生成包含镜像来源的列表文件
hangar generate-list --rancher="v2.7.0" --output-source ./generated-list-source.txt

# 使用 --output-windows 参数，指定输出的 Windows 镜像列表文件名称
# 默认情况下本工具不会单独生成 Windows 镜像列表文件
hangar generate-list --rancher="v2.7.0" --output-windows ./generated-list-windows.txt

# 使用 --dev 参数，在没有使用 --chart, --system-chart, --kdm 参数时，
# 自动从 KDM 和 chart 的 dev 分支生成镜像列表
# 默认情况下此工具会从 release 分支生成镜像列表
hangar generate-list --rancher="v2.7.0" --dev

# 使用 --chart 参数，指定 chart 仓库的路径
hangar generate-list --rancher="v2.7.0" --chart ./charts

# 使用 --system-chart 参数，指定 system-chart 仓库的路径
hangar generate-list --rancher="v2.7.0" --system-chart ./system-chart

# 使用 --kdm 参数，指定 KDM Data 文件的位置或 URL 链接
hangar generate-list --rancher="v2.7.0" --kdm ./data.json
hangar generate-list --rancher="v2.7.0" --kdm https://releases.rancher.com/kontainer-driver-metadata/release-v2.7/data.json

# 使用 --tls-verify=false 参数，跳过 URL 链接的 TLS 验证
hangar generate-list --rancher="v2.7.0" \
    --kdm "https://[insecure-https-url]/data.json" \
    --tls-verify=false

# 使用 --debug 参数，输出更详细的调试日志
hangar generate-list --rancher="v2.7.0" --debug
```

### 自定义 KDM 文件和 Chart 仓库

执行此工具时，如果只指定 `--rancher` 命令行参数，将自动根据 Rancher 版本获取 KDM 数据并克隆 Chart 仓库到本地。除此之外可通过 `--chart`、`--system-chart`、`--kdm` 参数自定义生成镜像列表时读取的 KDM 数据文件和 Chart 仓库。

> 在有多个 chart 和 system-chart 仓库需要加载时，可指定多个 `--chart` 和 `--system-chart` 参数。

```sh
# 首先下载 KDM data.json，克隆 chart 仓库到本地
hangar generate-list \
    --rancher="v2.7.0" \
    --kdm ./data.json \
    --chart ./charts-1 \
    --chart ./charts-2 \
    --system-chart ./system-charts-1 \
    --system-chart ./system-charts-2
```
