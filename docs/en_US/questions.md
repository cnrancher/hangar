# FAQ
> [简体中文](/docs/zh_CN/questions.md)

## Questions about Hangar

1. Principle of Mirror / Load / Save functions

    Hangar uses `skopeo` to copy container images from source registry server to destination registry server / local file.

    It requires `skopeo` installed when running Hangar.

## Common errors

1. Error `this tool does not support template version "va.b.c"`

    The Template Version saved in the compressed package does not match the version supported by the current version.

    Please ensure that the version of the tool when saving the image to the compressed package is the same as that of the tool when loading the compressed package by load.

    | Template Version | `hangar` version |
    | :--------------: | :---------------: |
    | `v1.0.0` | `v1.0.0` ~ `v1.2.3-rc1` |
    | `v1.1.0` | `v1.3.0` ~ latest |

2. Error `manifest unknown: manifest unknown'`

    `manifest unknown` means that the image was not found, please check the image in the image list.

    Try to use `skopeo inspect docker:// --raw | jq` to check whether the Manifest of the image can be obtained.

3. Error reporting `unsupported MIME type`

    The `mediaType` format of the Manifest of the source image is not supported.

    This tool supports the following types of `mediaType`:

    - `application/vnd.docker.distribution.manifest.list.v2+json`
    - `application/vnd.docker.distribution.manifest.v2+json`
    - `application/vnd.docker.distribution.manifest.v1+json`
    - `application/vnd.oci.image.manifest.v1+json`
    - `application/vnd.oci.image.index.v1+json`

    You can use `skopeo inspect docker:// --raw | jq` to get the `mediaType` of the Manifest.

4. Error `no image available for specified arch list`

    The architecture of the image to be copied does not match the architecture specified by the `-a` parameter.
