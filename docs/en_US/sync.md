# Sync
> [简体中文](/docs/zh_CN/sync.md)

The sync command syncs the extra container images into the cache folder created by [Save](./save.md) command.

## Background

There may some images failed to save when running Save command, the save failed image list will output into `save-failed.txt`. You can use Sync command to re-save these images into `saved-image-cache` folder and use [Compress](./compress.md) to re-compress the tarball.

Besides, the [decompress](./decompress.md) command supports to decompress the tarball created by hangar same as the decompress part of [hangar load](./load.md) command.

## QuickStart

Re-save the images in `saved-failed.txt` into `saved-images-cache` folder:

```sh
hangar sync -f ./saved-failed.txt -d ./saved-images-cache -j 10
```

> Sync failed images will output to `sync-failed.txt`

## Usage

```txt
Usage:
  hangar sync [flags]

Examples:
  hangar sync -f save-failed.txt -d [DECOMPRESSED_FOLDER]

Flags:
  -a, --arch string          architecture list of images, separate with ',' (default "amd64,arm64")
  -d, --destination string   decompressed saved images folder (required)
  -o, --failed string        file name of the sync failed image list (default "sync-failed.txt")
  -f, --file string          image list file (the format as same as 'rancher-images.txt') (required)
  -h, --help                 help for sync
  -j, --jobs int             worker number, concurrent mode if larger than 1, max 20 (default 1)
      --no-arch-os-fail      image copy failed when the OS and architecture of the image are not supported
      --os string            OS list of images, separate with ',' (default "linux,windows")
  -s, --source string        override the source registry defined in image list

Global Flags:
      --debug        enable debug output
      --tls-verify   enable https tls verify (default true)
```

## Others

After syncing images into cache folder, you can use [compress](./compress.md) command to create tarball.
