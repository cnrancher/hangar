# Decompress

Decompress the compressed tarball created by [save](./save.md) command.

## QuickStart

Decompress the `saved-images.tar.gz`.

```sh
hangar decompress -f ./saved-images.tar.gz
```

Same as Load command, The Decompress command can identify the spilted `.partX` file suffix.

```console
$ ls -alh
drwxr-xr-x   6 root  root   192B  1  6 18:00 .
drwxr-x---+ 70 root  root   2.2K  1  6 18:00 ..
-rw-r--r--   1 root  root    50M  1  6 17:59 saved-images.tar.gz.part0
-rw-r--r--   1 root  root    50M  1  6 17:59 saved-images.tar.gz.part1
-rw-r--r--   1 root  root    50M  1  6 17:59 saved-images.tar.gz.part2
-rw-r--r--   1 root  root   5.3M  1  6 17:59 saved-images.tar.gz.part3

$ hangar decompress -f saved-images.tar.gz
18:01:28 [INFO] decompressing saved-images.tar.gz...
18:01:28 [INFO] Read "saved-images.tar.gz.part0"
18:01:28 [INFO] Read "saved-images.tar.gz.part1"
18:01:28 [INFO] Read "saved-images.tar.gz.part2"
18:01:28 [INFO] Read "saved-images.tar.gz.part3"
......
```

## Usage
> [简体中文](/docs/zh_CN/decompress.md)

```txt
Usage:
  hangar decompress [flags]

Flags:
  -f, --file string     file name to be decompressed (required)
      --format string   compress format (available: 'gzip', 'zstd') (default "gzip")
  -h, --help            help for decompress

Global Flags:
      --debug        enable debug output
      --tls-verify   enable https tls verify (default true)
```
