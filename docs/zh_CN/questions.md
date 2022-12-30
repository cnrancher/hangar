# 常见问题

1. 报错 `this tool does not support template version "va.b.c"`

该工具 Load 时所加载的压缩包版本与当前工具所支持的版本不匹配。

2. 报错 `manifest unknown: manifest unknown"`

`manifest unknown` 表示没有找到该镜像，请检查镜像列表中的镜像。

尝试使用 `skopeo inspect docker://<image> --raw | jq` 检查是否能获取到该镜像的 Manifest。

3. 报错 `invalid media type`

源镜像的 Manifest 的 `mediaType` 格式不被支持。

本工具支持以下类型的 `mediaType`：

- `application/vnd.docker.distribution.manifest.list.v2+json`
- `application/vnd.docker.distribution.manifest.v2+json`

可使用 `skopeo inspect docker://<image> --raw | jq` 获取源镜像的 Manifest 的 `mediaType`。

4. 报错 `no image available for specified arch list`

待拷贝镜像的架构与 `-a` 参数指定的架构不匹配。
