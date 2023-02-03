# generate-list

```console
$ image-tools generate-list -h
Usage of generate-list:
  -chart value
        chart path
  -debug
        enable the debug output
  -dev
        Switch to dev branch/url of charts & KDM data
  -git-token string
        git token for download image data from RPM GC private repos
  -kdm string
        kdm path/url
  -kubeversion string
        minimum kuber version (semantic version with 'v' prefix) (default "v1.21.0")
  -o string
        generated image list path (linux and windows images) (default "generated-list.txt")
  -output-linux string
        generated linux image list
  -output-source string
        generate image list with image source
  -output-windows string
        generated windows image list
  -rancher string
        rancher version (semver with 'v' prefix)(use '-ent' suffix to distinguish GC)
  -registry string
        override the registry url
  -system-chart value
        system chart path
```

## QuickStart

According to the Rancher version, get the latest KDM data, and automatically clone the Chart repo to the local to generate a image-list:

```sh
./image-tools generate-list -rancher="v2.7.0"
```

## Parameters

Command-line parameters:

```sh
# Use the -rancher parameter to specify the Rancher version number.
# if only the Rancher version number is specified, the tool will automatically
# download the corresponding KDM data according to the Rancher version number,
# and clone the charts repo to the local, and generate an image list from them.
./image-tools generate-list -rancher="v2.7.0"

# Use the -registry parameter to specify the Registry URL of the generated image
# (the default is an empty string)
./image-tools generate-list -rancher="v2.7.0" -registry="docker.io"

# Use the -o parameter to specify the name of the output image list file
# (default is generated-list.txt)
./image-tools generate-list -rancher="v2.7.0" -o ./generated-list.txt

# Use the -output-linux parameter to specify the output Linux image-list file
# By default, this tool will not generate a separate Linux image-list file
./image-tools generate-list -rancher="v2.7.0" -output-linux ./generated-list-linux.txt

# Use the -output-source parameter to specify the output list file containing the image-source
# By default this tool will not generate a list file containing image-sources
./image-tools generate-list -rancher="v2.7.0" -output-source ./generated-list-source.txt

# Use the -output-windows parameter to specify the output Windows image-list file
# By default, this tool will not generate a Windows image-list file separately
./image-tools generate-list -rancher="v2.7.0" -output-windows ./generated-list-windows.txt

# Use the -dev parameter, automatically generate mirrorlists from dev branches
# of KDM and chart when not using the -chart, -system-chart, -kdm parameters,
# By default this tool will generate a list of mirrors from the release branch
./image-tools generate-list -rancher="v2.7.0" -dev

# Use the -kubeversion parameter to specify the minimum Kubernetes version (default v1.21.0)
./image-tools generate-list -rancher="v2.7.0" -kubeversion="v1.21.0"

# Use the -chart parameter to specify the path of the cloned chart repository
./image-tools generate-list -rancher="v2.7.0" -chart ./charts

# Use the -system-chart parameter to specify the path of the cloned system-chart repository
./image-tools generate-list -rancher="v2.7.0" -system-chart ./system-chart

# Use the -kdm parameter to specify the file path or URL of the KDM Data file
./image-tools generate-list -rancher="v2.7.0" -kdm ./data.json
./image-tools generate-list -rancher="v2.7.0" -kdm https://releases.rancher.com/kontainer-driver-metadata/release-v2.7/data.json

# Use the -debug parameter to output more detailed debug logs
./image-tools generate-list -rancher="v2.7.0" -debug
```

### Customize KDM data file and Chart repositories

When executing this tool, if only the `-rancher` command line parameter is specified,
KDM data will be automatically obtained according to the Rancher version and the Chart repo will be cloned to local automaitcally.

In addition, you can customize the KDM data file and Chart repository when generating the image-list
by using `-chart`, `-system-chart`, `-kdm` parameters.

> You can use multiple `-chart` and `-system-chart` parameters to specify multiple chart repos.

```sh
# Download KDM data.json and clone chart repository into local directory first.
./image-tools generate-list \
    -rancher="v2.7.0" \
    -kubeversion="v1.21.0" \
    -kdm ./data.json \
    -chart ./charts-1 \
    -chart ./charts-2 \
    -system-chart ./system-charts-1 \
    -system-chart ./system-charts-2
```

## Output

This tool will eventually generate a list file containing Windows and Linux images
from the Chart repository and KDM data. If you need to view the source of the images,
you can add an use `-output-source="FILE_NAME.txt"` parameter  to generate a list file
containing image sources.
