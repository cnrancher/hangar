# Decompress

Decompress 命令用来解压 Save 命令生成的压缩的压缩包文件。

## QuickStart

将 `saved-images.tar.gz` 文件解压。

```sh
hangar decompress -f ./saved-images.tar.gz
```

与 Load 命令在导入镜像时解压的方式一致，Decompress 命令支持识别分片压缩生成的 `.partX` 后缀的文件。

```console
$ ls -alh
drwxr-xr-x   6 root  root   192B  1  6 18:00 .
drwxr-x---+ 70 root  root   2.2K  1  6 18:00 ..
-rw-r--r--   1 root  root    50M  1  6 17:59 saved-images.tar.gz.part0
-rw-r--r--   1 root  root    50M  1  6 17:59 saved-images.tar.gz.part1
-rw-r--r--   1 root  root    50M  1  6 17:59 saved-images.tar.gz.part2
-rw-r--r--   1 root  root   5.3M  1  6 17:59 saved-images.tar.gz.part3

$ hangar decompress -f saved-images.tar.gz
18:01:28 [INFO] decompressing saved-images.tar.gz...
18:01:28 [INFO] Read "saved-images.tar.gz.part0"
18:01:28 [INFO] Read "saved-images.tar.gz.part1"
18:01:28 [INFO] Read "saved-images.tar.gz.part2"
18:01:28 [INFO] Read "saved-images.tar.gz.part3"
......
```

## Parameters

命令行参数：

```sh
# -f, --file 指定待解压的文件
hangar decompress -f ./saved-images.tar.gz

# --format 指定待解压的文件的压缩格式
# 可选: gzip, zstd
# 默认: gzip
hangar decompress -f ./saved-images.tar.zstd --format=zstd

# 使用 --debug 参数，输出更详细的调试日志
hangar decompress -f ./saved-images.tar.gz --debug
```
