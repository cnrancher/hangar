# Load

The Load command imports the saved file created by the [save](./save.md) command into the destination registry server, and creates manifest list for imported images.

## Quick Start

Load the `tar.gz` tarball created by the `save` command to `docker.io` registry server:

```sh
hangar load -s ./saved-images.tar.gz -d docker.io
```

Use `--compress` to specify the input file format when the not `tar.gz`.

```sh
# Load images from the decompressed cache directory
hangar load -s ./saved-image-cache -d docker.io --compress=dir

# Load images from tar.zstd tarball
hangar load -s ./saved-images.tar.zstd -d docker.io --compress=zstd
```

If the loaded files were splited into multi-parts by save command, the Load command will identify the `.partX` suffix automatically:

```console
$ ls -alh
drwxr-xr-x   6 root  root   192B  1  6 18:00 .
drwxr-x---+ 70 root  root   2.2K  1  6 18:00 ..
-rw-r--r--   1 root  root    50M  1  6 17:59 saved-images.tar.gz.part0
-rw-r--r--   1 root  root    50M  1  6 17:59 saved-images.tar.gz.part1
-rw-r--r--   1 root  root    50M  1  6 17:59 saved-images.tar.gz.part2
-rw-r--r--   1 root  root   5.3M  1  6 17:59 saved-images.tar.gz.part3

$ hangar load -s saved-images.tar.gz -d private.registry.io
18:01:28 [INFO] Decompressing saved-images.tar.gz...
18:01:28 [INFO] Read "saved-images.tar.gz.part0"
18:01:28 [INFO] Read "saved-images.tar.gz.part1"
18:01:28 [INFO] Read "saved-images.tar.gz.part2"
18:01:28 [INFO] Read "saved-images.tar.gz.part3"
......
```

### Harbor V2

If the destination image registry is Harbor V2, you can use the `--repo-type=harbor` parameter to automatically create the Harbor project (namespace).

If the image in the image list does not have Project defined during save (such as `mysql:8.0`, `busybox:latest`), then the `library` project will be automatically added to it during the Load process (`library/mysql:8.0`, `library/busybox:latest`).

You can use `--default-project=library` parameter to specify the name of the added Project (default is `library`).

## Usage

```txt
Usage:
  hangar load [flags]

Examples:
  hangar load -s SAVED_FILE.tar.gz -d REGISTRY_URL

Flags:
  -c, --compress string          compress format, can be 'gzip', 'zstd' or 'dir' (default "gzip")
      --default-project string   project name (also called 'namespace') when destination image project is empty (default "library")
  -d, --destination string       destination registry
  -o, --failed string            file name of the load failed image list (default "load-failed.txt")
      --harbor-https             use https when create harbor project (default true)
  -h, --help                     help for load
  -j, --jobs int                 worker number, concurrent mode if larger than 1, max 20 (default 1)
      --repo-type string         repository type, can be 'harbor' or empty
  -s, --source string            saved file to load (need to use '--compress' to specify the file format if not gzip)

Global Flags:
    --debug   enable debug output
```

## Load the splitted compressed files

```console
$ ls -alh
drwxr-xr-x 6 root root 192B 1 6 18:00 .
drwxr-x---+ 70 root root 2.2K 1 6 18:00 ..
-rw-r--r-- 1 root root 50M 1 6 17:59 saved-images.tar.gz.part0
-rw-r--r-- 1 root root 50M 1 6 17:59 saved-images.tar.gz.part1
-rw-r--r-- 1 root root 50M 1 6 17:59 saved-images.tar.gz.part2
-rw-r--r-- 1 root root 5.3M 1 6 17:59 saved-images.tar.gz.part3

$ hangar load -s saved-images.tar.gz -d private.registry.io
18:01:28 [INFO] Decompressing saved-images.tar.gz...
18:01:28 [INFO] Read "saved-images.tar.gz.part0"
18:01:28 [INFO] Read "saved-images.tar.gz.part1"
18:01:28 [INFO] Read "saved-images.tar.gz.part2"
18:01:28 [INFO] Read "saved-images.tar.gz.part3"
…
```
