# Save

```console
$ ./image-tools save -h
Usage of save:
  -a string
        architecture list of images, seperate with ',' (default "amd64,arm64")
  -d string
        Output saved images into tar.gz (default "saved-images.tar.gz")
  -debug
        enable the debug output
  -f string
        image list file
  -j int
        job number, async mode if larger than 1, maximum is 20 (default 1)
  -o string
        file name of the save failed image list (default "save-failed.txt")
  -s string
        override the source registry
```

## Preperation

Use `docker login <registry-url>` to login manually before running `image-tools save`.

## image-list format

**"NAME:TAG"**, separated with `:`.

```txt
# <NAME>:<TAG>
rancher/rancher:v2.7.0
```

> The line begins with `#` or `//` will be treated as comment.

## QuickStart

Save the images in `rancher-images.txt` into `tar.gz` tarball.

```sh
./image-tools save -f ./rancher-images.txt -d saved-images.tar.gz
```

> It download all image files (blob) into `saved-image-cache` directory first, then create a `tar.gz` tarball for this directory.

## Parameters

Parameters available:

```sh
# Use -f (file) to specify the image list file.
./image-tools save -f ./list.txt

# Use -d (destination) to specify the saved tar.gz file name
# default is saved-images.tar.gz
./image-tools save -f ./list.txt -d saved-images.tar.gz

# Use -s (source) to specify the source image registry.
# default is docker.io
./image-tools save -f ./list.txt -s custom.registry.io -d saved-images.tar.gz

# Use -a (arch) to specify the architextures, separate with ','
# default is amd64,arm64
./image-tools save -f ./list.txt -a amd64,arm64 -d saved-images.tar.gz

# Use -j (jobs) to specify the jobs num for save images concurrently.
./image-tools save -f ./list.txt -d saved-images.tar.gz -j 10 # run 10 workers

# You can input image list manually without -f parameter.
# Concurrent mode is not supported in this mode.
# You should exit this program by using `Ctrd-D` instead of `Ctrl-C`
./image-tools save -d saved-images.tar.gz
......
>>> rancher/rancher:v2.7.0

# Using -o (output) to specify the save failed images list file name.
# default is save-failed.txt
./image-tools save -f image-list.txt -o failed-list.txt

# Use -debug to enable debug output
./image-tools save -debug
```

## Logs

The logs line has "TIME LEVEL" prefix, you can track image information by using `M_ID` and `IMG_ID` in concurrent mode.

## Output

This tool will output a `tar.gz` tarball and `saved-failed.txt` list if there are some images failed to save.
