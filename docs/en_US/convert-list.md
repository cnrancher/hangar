# convert-list

The `convert-list` command converts the format of image list file `rancher-images.txt` to the list file used by the [Mirror](./mirror.md) command.

## Quick Start

Convert the list format from `rancher-images.txt` into the format used for the [mirror](./mirror.md) command, and set the destination registry to `custom.private.io`:

```sh
hangar convert-list -i rancher-images.txt -d custom.private.io
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

## Usages

```txt
Usage:
  hangar convert-list [flags]

Examples:
  hangar convert-list -i rancher-images.txt -o CONVERTED_MIRROR_LIST.txt

Flags:
  -d, --destination string   specify the destination registry
  -h, --help                 help for convert-list
  -i, --input string         input image list (required)
  -o, --output string        output image list (default "[INPUT_FILE].converted")
  -s, --source string        specify the source registry

Global Flags:
      --debug   enable debug output
```
