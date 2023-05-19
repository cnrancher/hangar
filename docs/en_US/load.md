# Load
> [简体中文](/docs/zh_CN/load.md)

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

> Add `--harbor-https=false` when Harbor registry is HTTP.

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
    --debug        enable debug output
    --tls-verify   enable https tls verify (default true)
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

## Load from different architectures tarball

> Support from `v1.6.0`.

Example:

```sh
# example image list
cat list.txt
docker.io/library/nginx:1.22
docker.io/library/nginx:1.23

# save AMD64 architecture images only
hangar save -f list.txt -a "amd64" -d amd64-images.tar.gz

# save ARM64 arhitexture images only
hangar save -f list.txt -a "arm64" -d arm64-images.tar.gz
```

Load command supports importing `amd64-images.tar.gz` and `arm64-images.tar.gz` in this example to the registry server, and the finally generated manifest list contains two architectures.

```sh
# Load the AMD64 tarball to Registry Server firstly
hangar load -s amd64-images.tar.gz -d <REGISTRY_URL>

# The manifest list of loaded image only contains the ARM64 architecture
skopeo inspect docker://<REGISTRY_URL>/library/nginx:1.22 --raw | jq
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.docker.distribution.manifest.list.v2+json",
  "manifests": [
    {
      "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
      "size": 1235,
      "digest": "sha256:66f1a9ae96f5a18068fcbd53e0171c78b40adffa3d70f565341eb453a34bb099",
      "platform": {
        "architecture": "arm64",
        "os": "linux",
        "variant": "v8"
      }
    }
  ]
}

# Load the ARM64 image tarball to Registry Server
hangar load -s arm64-images.tar.gz -d <REGISTRY_URL>

# After importing the image tarball of the two architectures,
# the manifest list including AMD64 and ARM64 architectures
skopeo inspect docker://<REGISTRY_URL>/library/nginx:1.22 --raw | jq
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.docker.distribution.manifest.list.v2+json",
  "manifests": [
    {
      "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
      "size": 1235,
      "digest": "sha256:66f1a9ae96f5a18068fcbd53e0171c78b40adffa3d70f565341eb453a34bb099",
      "platform": {
        "architecture": "arm64",
        "os": "linux",
        "variant": "v8"
      }
    },
    {
      "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
      "size": 1235,
      "digest": "sha256:7dcde3f4d7eec9ccd22f2f6873a1f0b10be189405dcbfbaac417487e4fb44c4b",
      "platform": {
        "architecture": "amd64",
        "os": "linux"
      }
    }
  ]
}
```

