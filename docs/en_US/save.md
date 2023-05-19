# Save
> [简体中文](/docs/zh_CN/save.md)

Save images from registry server into local file (compressed tarball or directory) and can be used by [load](./load.md) command.

## Image List Format

Each line contains **"image name: TAG"**, and the image and TAG are separated by `:`, for example:

```txt
# <NAME>:<TAG>
rancher/rancher:v2.7.0
```

> Line starts with `#` or `//` will be treated as a comment.

## Quick Start

Download all the images in the `rancher-images.txt` and create a `tar.gz` archive:

```sh
hangar save -f ./rancher-images.txt -d saved-images.tar.gz
```

> This command will download the image to the `saved-image-cache` cache folder firstly, and then create a compressed tarball of this folder.

## Usage

```txt
Usage:
  hangar save [flags]

Examples:
  hangar save -f rancher-images.txt -j [WORKER_NUM] -d SAVED_FILE.tar.gz

Flags:
  -a, --arch string          architecture list of images, separate with ',' (default "amd64,arm64")
  -c, --compress string      compress format, can be 'gzip', 'zstd' or 'dir' (set to 'dir' to disable compression, rename the cache directory only) (default "gzip")
  -d, --destination string   file name of saved images
                             (can use '--compress' to specify the output file format, default is gzip)
                             (default "saved-images.[FORMAT_SUFFIX]")
  -o, --failed string        file name of the save failed image list (default "save-failed.txt")
  -f, --file string          image list file (the format as same as 'rancher-images.txt')
  -h, --help                 help for save
  -j, --jobs int             worker number, concurrent mode if larger than 1, max 20 (default 1)
      --part                 enable segment compress
      --part-size string     segment part size (number(Bytes), or a string with 'K', 'M', 'G' suffix) (default "2G")
  -s, --source string        override the source registry defined in image list

Global Flags:
      --debug        enable debug output
      --tls-verify   enable https tls verify (default true)
```

## Save principle

> **The compressed tarball created by the Save command is not compatible with the compressed package created by `docker save`.**

Hangar uses `skopeo copy` to save the image blobs in the image list to the local `saved-image-cache` folder in OCI format.

The blobs files will be saved into the `saved-image-cache/share` share folder.

After all images are saved, create a compressed package for `saved-image-cache`
(except for using `--compress=dir` parameter).

After creating the compressed package, the `saved-image-cache` folder will not be deleted automatically,
you can delete this folder to avoid disk usage.

## Split the compressed file into multi part

If you need to copy the generated tarball into a small capacity flash drive,
you can use the `--part` parameter to split the compressed tarball into multi-part,
and use the `--part-size` parameter to specify the size of each part (default is `2G`),
the size can be a number (byte) or a number with `K`, `M`, `G` suffix.

When `--part` option specified, the created tarball filename will ended with `.partX` suffix.

> The way to split tarball into multi-part is same with the Linux command `split`.

Here are some ways to combine splitted files and decompress it in command line.

```sh
# Combine all file parts
cat ./saved-images.tar.gz.part* > saved-images.tar.gz
# Decompress the combined compressed package
tar -zxvf ./saved-images.tar.gz

# Or use the following command to decompress with one command
cat ./saved-images.tar.gz.part* | tar -zxv
```

> You can use the `load` command with the `--compress=dir` parameter to load the image from the decompressed cache folder and upload it to the destination registry server.
