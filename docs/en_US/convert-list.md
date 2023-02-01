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
  -s string
        specify the source registry
```

## Quick Start

Convert the list format from `rancher-images.txt` into the format used for the `mirror` sub-command, and set the destination registry to `custom.private.io`

```sh
./image-tools convert-list -i rancher-images.txt -d custom.private.io
```

This command will convert the images in `rancher-images.txt` from format:

```txt
# NAME:TAG
rancher/rancher:v2.6.9
nginx
```

to the format used by `mirror` sub-command:

```txt
# SOURCE DEST TAG
rancher/rancher custom.private.io/rancher/rancher v2.6.9
nginx custom.private.io/nginx latest
```

## Examples

Command line parameters:
```sh
# Use -i (input) and -d (destination) parameters,
# Specify the input image list file name and the registry of the target image
./image-tools convert-list -i list.txt -d private.registry.io

# Use the -s (source) parameter to specify the source registry of the converted image list
./image-tools convert-list -i list.txt -s source.io -d dest.io

# Use the -o (output) parameter to specify the file name of the output mirror list
# By default, the .converted suffix is added to the input file name
./image-tools convert-list -i list.txt -o converted.txt
```
