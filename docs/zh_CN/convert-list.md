# convert-list

```console
$ hangar convert-list -h
Usage of convert-list:
  -d string
        specify the dest registry
  -i string
        input image list
  -o string
        output image list
  -s string
        specify the source registry
```

## QuickStart

将下载的 `rancher-images.txt` 镜像列表格式转换至 `mirror` 时输入的镜像列表格式，并设定目标镜像的 registry 为 `custom.private.io`：

``` sh
./hangar convert-list -i rancher-images.txt -d custom.private.io
```

此命令会将 `rancher-images.txt` 格式的镜像列表：

```txt
# NAME:TAG
rancher/rancher:v2.6.9
nginx
```

转换为 `mirror` 时输入的镜像列表格式：

```txt
# SOURCE DEST TAG
rancher/rancher custom.private.io/rancher/rancher v2.6.9
nginx custom.private.io/nginx latest
```

## Parameters

命令行参数：

```sh
# 使用 -i (input) 和 -d (destination) 参数，
# 指定输入的镜像列表文件名和目标镜像的 registry
./hangar convert-list -i list.txt -d private.registry.io

# 使用 -s (source) 参数指定转换格式后的镜像列表的源 registry
./hangar convert-list -i list.txt -s source.io -d dest.io

# 使用 -o (output) 参数，指定输出镜像列表的文件名
# 默认为输入的文件名添加 .converted 后缀
./hangar convert-list -i list.txt -o converted.txt
```
