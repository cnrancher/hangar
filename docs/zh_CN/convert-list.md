# convert-list

```console
$ image-tools convert-list -h
Usage of convert-list:
  -d string
        specify the dest registry
  -i string
        input image list
  -o string
        output image list
```

## QuickStart

将下载的 `rancher-images.txt` 镜像列表格式转换至 `mirror` 时输入的镜像列表格式，并设定目标镜像的 registry 为 `custom.private.io`：

``` sh
./image-tools convert-list -i list.txt -d custom.private.io
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
./image-tools convert-list -i list.txt -d private.registry.io

# 使用 -o (output) 参数，指定输出镜像列表的文件名
# 默认为输入的文件名添加 .converted 后缀
./image-tools convert-list -i list.txt -o converted.txt
```
