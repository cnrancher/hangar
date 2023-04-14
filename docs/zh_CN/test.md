# Test

此项目包含两种测试，分别为 Unit test 和 Validation test。

## Validation test

Validation Test 包含以下内容：
1. 测试 Hangar 输出 Version、Help 帮助信息等基本的测试。
1. 最先执行 mirror, mirror-validate 命令的测试，将一些容器镜像 Mirror 至新部署的 Harbor Registry 服务器，并验证 Mirror 过的镜像是否正确。
    > 先将容器镜像从第三方 Public Registry Mirror 至 Harbor Private Registry，之后的 Save/Load 等测试将依靠 Harbor Private Registry 进行，以避免触发 DockerHub 的 Rate Limit 限制。
1. 之后执行 save, load, load-validate, sync, compress, decompress 等命令的测试。

### 测试环境准备

1. 搭建 Harbor V2 Registry 服务器，测试时会将容器镜像 Mirror 至 Harbor 仓库中，避免触发 Docker Hub Rate Limit 限制。
1. 设置环境变量，在测试时登录至 Docker Hub 和 Harbor V2。
    ```sh
    export SOURCE_REGISTRY="" # 源 Registry 设置为空字符串
    export SOURCE_USERNAME="" # DockerHub 用户名 (可选)
    export SOURCE_PASSWORD="" # DockerHub 密码 (可选)

    export DEST_REGISTRY="" # harbor registry url
    export DEST_USERNAME="" # harbor 用户名
    export DEST_PASSWORD="" # harbor 密码
    ```
1. 在本工程的根目录中执行 `make build` 编译生成可执行文件，供测试使用。

### 在容器中运行测试代码

可使用以下命令，在容器中一键运行所有子命令的 Validation test：

```console
$ make test_all
```

除此之外，可执行 `make test_[COMMAND_NAME]`，在容器中运行不同子命令的 Validation test：

> 此时需要先执行 `make test_mirror`，之后再执行其他命令的测试。

```sh
# 测试 mirror | mirror-validate 子命令
make test_mirror

# 测试 save 子命令
make test_save

# 测试 load | load-validate 子命令
make test_load

# 测试 sync | compress | decompress 子命令
make test_sync

# 以此类推，运行其他命令的测试指令为：
make test_[COMMAND_NAME]
```

## Unit test

单元测试用来确保程序代码中函数的执行和输出是否符合预期。

在容器中运行程序代码的单元测试：

```console
$ make test
```
