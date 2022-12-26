# 自建 SSL Certificate

如果镜像仓库为使用自建 SSL Certificate 的私有镜像仓库 (自签名 Harbor)，请参照以下部分进行配置。

## Docker

> FYI: [Use self-signed certificates](https://docs.docker.com/registry/insecure/#use-self-signed-certificates) (从步骤3开始)

将 SSL 公钥拷贝至 `/etc/docker/certs.d/<registry-url>/ca.crt`。

```console
# sudo mkdir -p /etc/docker/certs.d/${REGISTRY_URL}/
# cp public.crt /etc/docker/certs.d/${REGISTRY_URL}/ca.crt
```

## Docker Buildx

> FYI: <https://github.com/docker/buildx/issues/80#issuecomment-533844117>

本工具使用 `docker-buildx` 创建 Manifest List，在使用自建 SSL 时需要将公钥粘贴至
`/etc/ssl/certs/ca-certificates.crt` 文件内。

```console
# cat >> /etc/ssl/certs/ca-certificates.crt <<'EOF'
-----BEGIN CERTIFICATE-----
...
-----END CERTIFICATE-----
EOF
```
