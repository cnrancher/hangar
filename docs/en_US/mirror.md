# Mirror

## Image List Format

> The image list format used by `mirror` and `mirror-validate` command are different from `rancher-images.txt`. You can use [convert-list](./convert-list.md) command to convert the image list format.

Each line contains **"[source image] [destination image] [TAG]"**, separated by spaces:

```txt
# <SOURCE> <DEST> <TAG>
docker.io/hello-world private.io/library/hello-world latest
```

The registry URL of images is not force required and can be empty:

```txt
# <SOURCE> <DEST> <TAG>
hello-world library/hello-world latest
```

> Line starts with `//` or `#` will be treated as a comment.

## Quick Start

Mirror images in the `image-list.txt`, use `-f` to specify the image list file name, and `-d` to specify the destination registry URL.

```sh
hangar mirror -f ./image-list.txt -d <DESTINATION_REGISTRY_URL>
```

### Harbor V2

If the destination image registry is Harbor V2, you can use the `--repo-type=harbor` parameter to automatically create the Harbor project (namespace).

If the image in the image list does not have Project defined during Save (such as `mysql:8.0`, `busybox:latest`), then the `library` project will be automatically added to it during the Load process (`library/mysql:8.0`, `library/busybox:latest`).

You can use `--default-project=library` parameter to specify the name of the added Project (default is `library`).

## Usage

```txt
Usage:
  hangar mirror [flags]

Examples:
  hangar mirror -f MIRROR_IMAGE_LIST.txt -s SOURCE_REGISTRY -d DEST_REGISTRY

Flags:
  -a, --arch string              architecture list of images, separate with ',' (default "amd64,arm64")
      --default-project string   project name (also called 'namespace') when destination image project is empty (default "library")
  -d, --destination string       override the destination registry defined in image list
  -o, --failed string            file name of the mirror failed image list (default "mirror-failed.txt")
  -f, --file string              image list file (should be 'mirror' format)
      --harbor-https             use https when create harbor project (default true)
  -h, --help                     help for mirror
  -j, --jobs int                 worker number, concurrent mode if larger than 1 (default 1)
      --repo-type string         repository type of dest registry server (can be 'harbor' or empty string)
  -s, --source string            override the source registry defined in image list

Global Flags:
      --debug   enable debug output
```
