# mirror-validate
> [简体中文](/docs/zh_CN/mirror-validate.md)

`mirror-validate` for validating the mirrored container images.

## Quick Start

After mirror images, verify the mirrored images to ensure that the images have been mirrored to the destination registry,
the validate failed images will output into `mirror-validate-failed.txt`.

**The input image list format should be same as the format used by the [Mirror](./mirror.md) command.**

```sh
hangar mirror-validate -f ./list.txt -j 10
```

## Usage

```txt
Usage:
  hangar mirror-validate [flags]

Examples:
  hangar mirror-validate -f MIRROR_IMAGE_LIST.txt -s SOURCE -d DESTINATION

Flags:
  -a, --arch string              architecture list of images, separate with ',' (default "amd64,arm64")
      --default-project string   project name (also called 'namespace') when destination image project is empty (default "library")
  -d, --destination string       override the destination registry defined in image list
  -o, --failed string            file name of the mirror validate failed image list (default "mirror-validate-failed.txt")
  -f, --file string              image list file (should be 'mirror' format)
  -h, --help                     help for mirror-validate
  -j, --jobs int                 worker number, concurrent mode if larger than 1, max 20 (default 1)
      --os string                OS list of images, separate with ',' (default "linux,windows")
  -s, --source string            override the source registry defined in image list

Global Flags:
      --debug        enable debug output
      --tls-verify   enable https tls verify (default true)
```

# FAQ

1. Error: `Validate failed: destination manifest MIME type unknown: application/vnd.docker.distribution.manifest.v2+json`.

      This error will occur when the destination image MediaType is not `"application/vnd.docker.distribution.manifest.list.v2+json"`.

      You can use `skopeo inspect docker://<dest-image>:<tag> --raw` to check the MediaType type of the destination image.

2. Error: `destination manifest does not exists`, indicates that the destination image does not exist, please check the destination image.

3. Error: `destination manifest list length should be 1` indicates that the Manifest of the source image contains only one image, so there should be only one image in the Manifest List of the destination image. If there are multiple images in the Manifest List of the destination image, this error will appear.

      You can use `skopeo inspect docker://<dest-image>:<tag> --raw` to view the Manifest List of the destination image.

4. Error: `source * != dest *` indicates that some information of the source image does not match the destination image, such as Arch, Variant, OS, etc.

5. Encountered the following error:

      ```text
      11:22:33 [ERRO] [M_ID:1] srcSpec: [
      {
            "digest": "sha256:9997c2f450f51e5c5402854899c42354b7968ca8298815df812b00409533527c",
            "platform": {
                  "architecture": "amd64",
                  "os": "linux"
            }
      }
      ]
      11:22:33 [ERRO] [M_ID:1] dstSpec: [
      {
            "digest": "sha256:8ace038ea3a18057e865b81e5ccd12d75ddeec0fdbd331555d877d39ac3f45bb",
            "platform": {
                  "architecture": "amd64",
                  "os": "linux"
            }
      }
      ]
      ```

      Indicates that the Manifest List of the source image (srcSpec) does not match the Manifest List of the destination image (dstSpec). If the `digest` does not match, it means that the upstream image has been updated, and the image in the private registry has not been updated. You can re-run `mirror ` command; if other fields do not match (`variant`, `os.version`), etc., you can also try to fix it by re-running the `mirror` command.
