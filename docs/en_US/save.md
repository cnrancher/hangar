# Save

```console
$ ./hangar save -h
Usage of save:
  -a string
        architecture list of images, separate with ',' (default "amd64,arm64")
  -compress string
        compress format, can be 'gzip', 'zstd' or 'dir' (default "gzip")
  -d string
        Output saved images into destination file (can use '-compress' to specify the output file format, default is gzip) (default "saved-images.tar.gz")
  -debug
        enable the debug output
  -f string
        image list file
  -j int
        job number, async mode if larger than 1, maximum is 20 (default 1)
  -o string
        file name of the save failed image list (default "save-failed.txt")
  -part
        enable segment compress
  -part-size string
        segment part size (number, or a string with 'K','M','G' suffix) (default "2G")
  -s string
        override the source registry
```

## Preparation

Before executing `hangar save`, if there is a private image in the image list, please make sure to manually login to the repo via `docker login`.

## Mirror List Format

Each line contains **"image name: TAG"**, and the image and TAG are separated by `:`, for example:

```txt
# <NAME>:<TAG>
rancher/rancher:v2.7.0
```

> If the line starts with `#` or `//`, then that line will be treated as a comment.

## Quick Start

Download all the images in the `rancher-images.txt` list on your local filesystem and create a `tar.gz` archive:

```sh
./hangar save -f ./rancher-images.txt -d saved-images.tar.gz
```

> This command will first download the image to the `saved-image-cache` cache folder, and then create a compressed package of this folder.

## Parameters

Usage examples & command line parameters:

```sh
# Use the -f (file) parameter to specify the image list file
./hangar save -f ./list.txt

# Use the -d (destination) parameter to specify the file name of the exported image
# Can be used with the -compress parameter
# The default file name is saved-images.tar.gz
./hangar save -f ./list.txt -d saved-images.tar.gz

# Use the -s (source) parameter to specify the registry of the source mirror without modifying the mirror list
# If the source image in the image list does not specify registry, and the -s parameter is not set, then the registry of the source image will by default be set to docker.io
./hangar save -f ./list.txt -s custom.registry.io -d saved-images.tar.gz

# Use the -a (arch) parameter to specify the architecture of the exported image (separated by commas)
# The default is amd64, arm64
./hangar save -f ./list.txt -a amd64,arm64 -d saved-images.tar.gz

# Use the -j (jobs) parameter to specify the number of concurrent workers to download images concurrently (1~20 jobs are supported)
./hangar save -f ./list.txt -d saved-images.tar.gz -j 10 # Start 10 workers

# Use the -part parameter to enable volume compression, the default size of each volume is 2G
# You can use the -part-size parameter to set the volume size
# After enabling sub-volume compression, a file with the suffix .part* will be created
./hangar save -f ./list.txt -d saved-images.tar.gz -part -part-size=4G # Specify the size of each volume to be 4G

# When the -f parameter is not set, you can manually enter the mirror list line by line to download a certain image
# Concurrent copying will not be supported in this mode
# Note that in this mode, use `Ctrl-D` to end the input of the image list, do not use `Ctrl-C` to end the program, otherwise the compressed package will not be created!
./hangar save -d saved-images.tar.gz
......
>>> rancher/rancher:v2.7.0

# Use the -o (output) parameter to output the list of images that failed to be saved to disk
# Default output will be saved to save-failed.txt
./hangar save -f image-list.txt -o failed-list.txt

# Use the -compress parameter to specify the compression format
# Optional: gzip, zstd, dir
# The default is gzip format, if dir format is specified, it means only save the image in the folder without compressing it
./hangar save -f image-list.txt -compress=zstd -d saved.tar.zstd

# Use the -debug parameter to output more detailed debug logs
./hangar save -debug
```

## Save principle

**The compressed package created by the Save command of this tool is not compatible with the compressed package created by `docker save`. **

When this tool executes Save, it first uses `skopeo copy` to save the image blobs in the image list to the local `saved-image-cache` folder in OCI format.

The image blobs files will be saved to the `saved-image-cache/share` shared folder.

After all images are downloaded, this tool will create a compressed package for `saved-image-cache` (except for `-compress=dir` parameter).

After creating the compressed package, this tool will not automatically delete the `saved-image-cache` folder, please manually delete this folder to save hard disk space.

## Volume compression

You can use the `-part` parameter to enable sub-volume compression, and use the `-part-size` parameter to specify the volume size, which supports absolute numbers (byte size) or with `K`, `M`, `G` suffix at the end specifying the unit.

When partition compression is enabled, the created archive will end with `.part*` suffix.

The principle of sub-volume compression created by this tool is consistent with the Linux command `split`. In addition to using the `load` command of this tool to decompress, you can also use the following command to decompress the sub-volume compressed package:

```sh
# Combine all volumes
cat ./saved-images.tar.gz.part* > saved-images.tar.gz
# Decompress the integrated compressed package
tar -zxvf ./saved-images.tar.gz

# Or use the following command to decompress with one command
cat ./saved-images.tar.gz.part* | tar -zxv
```

> You can use the `load` command of the image push tool with the `-compress=dir` parameter to load the image from the decompressed cache folder and upload it to the private registry.

## Logs

The log output emits "time, log level", `M_ID` (corresponding to the Nth Manifest list in the mirror list) and `IMG_ID` of each line of the log when the image is copied concurrently can be used to track which image failed to download.

## Output

This tool will eventually generate a `tar.gz` tarball.

If a certain image fails to be copied during the copy process, the tool will output the failed image list to `save-failed.txt`, and the `-o` parameter can be used to set the file name of the failed image list.
