# 常见问题

1. 报错 `this tool does not support template version "va.b.c"`

    该工具 Load 时所加载的压缩包中保存的 Template Version 与当前工具所支持的版本不匹配。

    请确保 Save 镜像至压缩包时工具的版本与 Load 加载压缩包时工具的版本一致。

    | Template Version | `image-tools` 版本 |
    | :--------------: | :---------------: |
    | `v1.0.0`         | `v1.0.0` ~ `v1.2.3-rc1` |
    | `v1.1.0`         | `v1.3.0` ~ latest |

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

5. 报错 `failed to read from destination repository <IMAGE>: 400 (Bad Request)`

    此错误由 Harbor 产生，在 Mirror / Load 目标镜像时没有指明 Project。

    可使用 `-repo-type=harbor` 参数设定目标镜像仓库的类型为 Harbor，并使用 `-default-project=library` 参数设定默认创建的 Project 名称。
    当目标镜像没有指明 Project 时，会自动为镜像名称添加 `library` Project。
