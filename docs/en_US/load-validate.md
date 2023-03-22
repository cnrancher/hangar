# load-validate

The `load-validate` command will verify the loaded image after executing `load` command to ensure that the image has been loaded to the destination registry. The list of images that fail the verification will be saved into the `load-validate-failed.txt` file.

## Quick Start

The input file is created by save command, and use `-d` to specify the destination registry URL.

```sh
hangar load-validate -s ./saved-images.tar.gz -d private.registry.io
```

You can use `--compress=dir` and `-s ./saved-image-cache` to specify the decompressed cache directory to load-validate.

```sh
hangar load-validate -s ./saved-image-cache -d private.registry.io --compress=dir
```

## Usage

```txt
Usage:
  hangar load-validate [flags]

Examples:
  hangar load-validate -s SAVED_FILE.tar.gz -d REGISTRY_URL

Flags:
  --compress string          compress format, can be 'gzip', 'zstd', or 'dir' (default "gzip")
  --default-project string   project name (also called 'namespace') when destination image project is empty (default "library")
  -d, --destination string       destination regitry
  -o, --failed string            file name of the validate failed image list (default "load-validate-failed.txt")
  -h, --help                     help for load-validate
  -j, --jobs int                 worker number, concurrent mode if larger than 1, max 20 (default 1)
  -s, --source string            saved file to load validate (need to use '--compress' to specify the file format if not gzip)

Global Flags:
  --debug   enable debug output
```

# FAQ

1. Error: `Validate failed: destination manifest MIME type unknown: application/vnd.docker.distribution.manifest.v2+json`.

      This error will occur when the MediaType of the Manifest of the destination image is not `"application/vnd.docker.distribution.manifest.list.v2+json"`.

      You can use `skopeo inspect docker://<dest-image>:<tag> --raw` to check the `MediaType` type of the Manifest of the destination image.

1. Error: `destination manifest does not exists`, indicates that the destination image does not exist, please check the destination image.

1. Encountered the following error:

      ```text
      11:22:33 [ERRO] [M_ID:1] srcSpec: [
      {
            "digest": "",
            "platform": {
                  "architecture": "amd64",
                  "os": "linux"
            }
      }
      ]
      11:22:33 [ERRO] [M_ID:1] dstSpec: [
      {
            "digest": "",
            "platform": {
                  "architecture": "amd64",
                  "os": "windows"
                  "os.version": "1.0.10"
            }
      }
      ]
      ```

      Indicates that the local image (srcSpec) does not match some fields of the server image (dstSpec).
