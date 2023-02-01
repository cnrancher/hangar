#mirror-validate
```console
$ image-tools mirror-validate -h
Usage of mirror-validate:
  -a string
        architecture list of images, separate with ',' (default "amd64,arm64")
  -d string
        override the destination registry
  -debug
        enable the debug output
  -f string
        image list file
  -j int
        job number, async mode if larger than 1, maximum is 20 (default 1)
  -o string
        file name of the validate failed image list (default "mirror-validate-failed.txt")
  -s string
        override the source registry
```

## Quick Start

After executing the `mirror` command, verify the mirrored image to ensure that the image has been mirrored to the target registry, and the list of images that failed the verification will be saved in the `mirror-validate-failed.txt` file.

The image list format entered should be equal to the image list format supported by the [Mirror](./mirror.md) subcommand. 

```sh
./image-tools mirror-validate -f ./image-list.txt -j 10
```

## Parameters

Command line parameters:
```sh
# Use -f (file) to specify the image list file
./image-tools mirror-validate -f ./list.txt

# Use the -d (destination) parameter to set the target image registry
./image-tools mirror-validate -f ./list.txt -d private.registry.io

# Use the -s (source) parameter to set the source image registry
./image-tools mirror-validate -f ./list.txt -s docker.io

# Use the -a (arch) parameter to set the architecture of the image (separated by commas)
# The default is amd64, arm64
./image-tools mirror-validate -f ./list.txt -a amd64,arm64,arm

# Use the -j (jobs) parameter to set the number of goroutine pools and concurrently verify the image (support 1~20 jobs)
./image-tools mirror-validate -f ./list.txt -j 20 # Start 20 Workers

# When the -f parameter is not set, you can manually enter the image list by line to verify a certain image
# Concurrency verification is not supported at this time
./image-tools mirror-validate
......
>>> hello-world library/hello-world latest

# Use the -o (output) parameter to output the image list that fails the verification to the specified file
# Default output to mirror-validate-failed.txt
./image-tools mirror-validate -f image-list.txt -o validate-failed-list.txt

# Use the -debug parameter to output more detailed debug logs
./image-tools mirror-validate -f ./list.txt -debug
```

# FAQ

Errors and reasons that may be encountered when using the verification function:

1. Error: `Validate failed: destination manifest MIME type unknown: application/vnd.docker.distribution.manifest.v2+json`. This error will occur when the MediaType of the Manifest of the target image is not `"application/vnd.docker.distribution.manifest.list.v2+json"`.

You can use `skopeo inspect docker://<dest-image>:<tag> --raw` to check the MediaType type of the manifest of the target image.

2. Error: `destination manifest does not exists`, indicates that the target image does not exist, please check the target image.

3. Error: `destination manifest list length should be 1` indicates that the Manifest of the source image contains only one image, so there should be only one image in the Manifest List of the target image. If there are multiple images in the Manifest List of the target image, this error will appear.

You can use to skopeo inspect docker://<dest-image>:<tag> --rawview the Manifest List of the target image.

4. Error: `source * != dest *` indicates that some information of the source image does not match the target image, such as Arch, Variant, OS, etc.

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

# Logs

The log output by executing this tool includes "time and log level". When verifying images concurrently, each line of logs M_ID(corresponding to the Nth Manifest list in the image list) can be used to track which image verification failed.

## Output

If a certain image fails to be verified during the verification process, the tool will output the list of images that failed to be verified to mirror-validate-failed.txt, and the file name of the image list that failed to be verified can be set with -oparameters .
