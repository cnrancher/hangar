# load-validate

```console
$ hangar load-validate -h
Usage of load-validate:
  -compress string
        compress format, can be 'gzip', 'zstd' or 'dir' (default "gzip")
  -d string
        target private registry: port
  -debug
        enable the debug output
  -default-project string
        project name when project is empty (default "library")
  -j int
        job number, async mode if larger than 1, maximum is 20 (default 1)
  -o string
        file name of the validate failed image list (default "load-validate-failed.txt")
  -s string
        saved file to load (tar tarball or a directory)
```

## Quick Start

After executing the `load` command, verify the image that has been loaded to ensure that the image has been loaded to the target registry, and the list of images that failed the verification will be saved in the `load-validate-failed.txt` file.

The input file is the compressed package file saved by the save subcommand or the directory name of the folder after decompression.

```sh
./hangar load-validate -s ./saved-images.tar.gz -d private.registry.io
```

## Parameters

Command line parameters:

```sh
# Use the -s (source) parameter to set the file name saved by save
# Use the -d (destination) parameter to set the target registry
./hangar load-validate -s ./saved-images.tar.gz -d private.registry.io

# Use the -j (jobs) parameter to set the number of coroutine pools and concurrently verify the image (support 1~20 jobs)
./hangar load-validate -s ./saved-images.tar.gz -d private.registry.io -j 10 # Start 10 workers

# Use the -compress parameter to specify the compression type of the imported file
# Optional: gzip, zstd, dir
# The default is the gzip format, if dir format is specified, it means loading the image from the folder for verification, and not decompressing it
./hangar load-validate -s ./saved-image-cache -d private.registry.io -compress=dir

# Use the -default-project parameter to specify the default project name
# The default value is library
# This parameter will rename `private.io/mysql:5.8` to `private.io/library/mysql:5.8`
./hangar load-validate -s ./saved-image-cache -d private.registry.io -default-project=library

# Use the -o (output) parameter to output the image list that fails the verification to the specified file
# Default output to load-validate-failed.txt
./hangar load-validate -s ./saved-images.tar.gz -d private.registry.io -o failed.txt

# Use the -debug parameter to output more detailed debug logs
./hangar load-validate -s ./saved-images.tar.gz -d private.registry.io -debug
```

# FAQ

Errors and issues that may be encountered when using the verification function:

1. Error: `Validate failed: destination manifest MIME type unknown: application/vnd.docker.distribution.manifest.v2+json`. This error will occur when the MediaType of the Manifest of the target image is not `"application/vnd.docker.distribution.manifest.list.v2+json"`.

You can use to skopeo inspect docker://<dest-image>:<tag> --rawcheck the MediaType type of the Manifest of the target image.

1. Error: `destination manifest does not exists`, indicates that the target image does not exist, please check the target image.

2. Encountered the following error:

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

# Logs

The log output by executing this tool includes "time, log level", and the `M_ID` of each line of log while verifying image concurrently can be used to track which image verification failed.

## Output

If a certain image fails to be verified during the verification process, the tool will output the list of failed images to `load-validate-failed.txt`, and the `-o` parameter can be used to set the list of images that failed to be verified the file name of the .
