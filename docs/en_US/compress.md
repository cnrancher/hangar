# Compress

Use compress command to compress the cache folder.

## QuickStart

Compress the `saved-image-cache` folder to `tar.gz` format tarball file.

```sh
hangar compress -f ./saved-image-cache
```

You can use `--format` parameter to specify the compress format (same as `--compress` parameter in [Save](./save.md) command).

```sh
hangar compress -f ./saved-image-cache --format=zstd
```

You can use `--part` and `--part-size` parameter to split the compressed file into multi-parts.

```sh
# Split files by 4GB
hangar compress -f ./saved-image-cache --part --part-size=4G
```

## Usage

```txt
Usage:
  hangar compress [flags]

Flags:
  -d, --destination string   file name of saved images
                             (can use '--compress' to specify the output file format, default is gzip)
                             (default "saved-images.[FORMAT_SUFFIX]")
  -f, --file string          saved image cache folder (required)
      --format string        compress format (available: 'gzip', 'zstd') (default "gzip")
  -h, --help                 help for compress
      --part                 enable segment compress
      --part-size string     segment part size (number(Bytes), or a string with 'K', 'M', 'G' suffix) (default "2G")

Global Flags:
      --debug   enable debug output
```