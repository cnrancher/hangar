# Mirror

```console
$ ./image-tools mirror -h
Usage of mirror:
  -a string
        architecture list of images, separate with ',' (default "amd64,arm64")
  -d string
        override the destination registry
  -debug
        enable the debug output
  -default-project string
        project name when dest repo type is harbor and dest project is empty (default "library")
  -f string
        image list file
  -harbor-https
        use HTTPS by default when create harbor project (default true)
  -j int
        job number, async mode if larger than 1, maximum is 20 (default 1)
  -o string
        file name of the mirror failed image list (default "mirror-failed.txt")
  -repo-type string
        repository type, can be 'harbor' or empty
  -s string
        override the source registry
```
## image List Format

Each line contains **"source image target image TAG"**, separated by spaces, for example:

```txt
# <SOURCE> <DEST> <TAG>
docker.io/hello-world private.io/library/hello-world latest
```

The source image and the target image can be images that do not contain the registry prefix, for example:

```txt
# <SOURCE> <DEST> <TAG>
hello-world library/hello-world latest
```

> If the line starts with `#` or `//`, then this line will be treated as a comment

## Quick Start

Mirror all the images in the `image-list.txt` list, use the `-f` parameter to specify the name of the image list, and `-d` to specify the target registry

```sh
./image-tools mirror -f ./image-list.txt -d
```

### Harbor V2

If the target image registry type is Harbor V2, then you can use the `-repo-type=harbor` parameter, which will automatically create a project for the Harbor V2 registry.

In addition, if the target image in the image list does not contain `Project` (such as `mysql:8.0`, `busybox:latest` of Docker Hub), then the `library` Project prefix will be automatically added to it during the mirror process (`library/mysql:8.0`, `library/busybox:latest`).

You can use `-default-project=library` parameter to specify the name of the added Project (default is `library`).

## Parameters

Command line parameters:

```sh
# Use the -f (file) parameter to specify the image list file
./image-tools mirror -f ./list.txt

# Use the -d (destination) parameter to specify the registry of the target image without modifying the image list
# If the target image in the list does not have a registry, and the -d parameter and the DOCKER_REGIRTSY environment variable are not set, then the registry of the target image will be set to the default docker.io
# The priority is: -d parameter > DOCKER_REGISTRY environment variable > registry written in the image list
./image-tools mirror -f ./list.txt -d private.registry.io

# Use the -s (source) parameter to specify the registry of the source image without modifying the image list
# If the source image in the image list does not write registry, and the -s parameter is not set, then the registry of the source image will be set to the default docker.io
./image-tools mirror -f ./list.txt -s docker.io

# Use the -a (arch) parameter to set the architecture of the copy image (separated by commas)
# The default is amd64, arm64
./image-tools mirror -f ./list.txt -a amd64,arm64

# Use the -j (jobs) parameter to specify the number of goroutine pools and copy images concurrently (support 1~20 jobs)
./image-tools mirror -f ./list.txt -j 10 # Start 10 workers

# When the -f parameter is not set, you can manually enter the image list by line to copy a certain image
# Concurrent copying will not be supported at this time
./image-tools mirror
.......
>>> hello-world library/hello-world latest

# Use -repo-type to specify the type of the target image registry, the default is an empty string, and it can be set to "harbor"
# When the type of the target image registry is harbor, a project will be automatically created for the target image
./image-tools mirror -f ./list.txt -repo-type=harbor

# Use the -default-project parameter to specify the default project name
# The default value is library
# This parameter will rename `private.io/mysql:5.8` to `private.io/library/mysql:5.8`
./image-tools mirror -f ./list.txt -repo-type=harbor -default-project=library

# Use the -o (output) parameter to output the list of failed images to the specified file
# Default output to mirror-failed.txt
./image-tools mirror -f image-list.txt -o failed-list.txt

# Use the -debug parameter to output more detailed debug logs
./image-tools mirror -debug
```

## Logs

The log output by executing this tool includes "time, log level", `M_ID` (corresponding to the Nth Manifest list in the image list) and `IMG_ID` of each line of the log when the image is copied concurrently can be used to track exactly which image copy failed.

## Output

If a certain image fails to be copied during the copy process, the tool will output the failed mirror list to `mirror-failed.txt`, and the `-o` parameter can be used to set the file name of the failed mirror list.
