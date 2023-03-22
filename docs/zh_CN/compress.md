# Compress

Compress 命令用来压缩 Save 命令生成的未压缩的 image 缓存文件夹。

## QuickStart

将 `saved-image-cache` 压缩为 `tar.gz` 格式的压缩包文件。

```sh
hangar compress -f ./saved-image-cache
```

可使用 `--format` 参数指定压缩文件格式（用法等同于 [Save](./save.md) 命令的 `--compress`）：

```sh
hangar compress -f ./saved-image-cache --format=zstd
```

可使用 `--part` 和 `--part-size` 参数启用分片压缩功能（参数的用法等同于 [Save](./save.md) 命令）

```sh
# 将压缩文件以 4G 为单位进行分割
hangar compress -f ./saved-image-cache --part --part-size=4G
```

## Parameters

命令行参数：

```sh
# -f, --file 指定待压缩的文件夹目录
## 若文件夹名称不是 saved-image-cache, 工具会先为文件夹重命名再压缩
hangar compress -f ./saved-image-cache

# --format 指定压缩格式
# 可选: gzip, zstd
# 默认: gzip
hangar compress -f ./saved-image-cache --format=zstd

# --part 启用分片压缩（将压缩文件按一定大小进行分割）
# --part-size 指定分片压缩的大小（默认为 2G）
hangar compress -f ./saved-image-cache --part --part-size=4G

# 使用 --debug 参数，输出更详细的调试日志
hangar compress -f ./saved-image-cache --debug
```