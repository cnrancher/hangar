# self-signed certificate

Using self-signed certificate.

## Docker

> FYI: [Use self-signed certificates](https://docs.docker.com/registry/insecure/#use-self-signed-certificates)

Copy the pulic key to `/etc/docker/certs.d/<registry-url>/ca.crt`.

```console
# sudo mkdir -p /etc/docker/certs.d/${REGISTRY_URL}/
# cp public.crt /etc/docker/certs.d/${REGISTRY_URL}/ca.crt
```

## Docker Buildx

> FYI: <https://github.com/docker/buildx/issues/80#issuecomment-533844117>

`image-tools` uses `docker-buildx` for creating manifest list.
You need to paste the pubkey into `/etc/ssl/certs/ca-certificates.crt`.

```console
# cat >> /etc/ssl/certs/ca-certificates.crt <<'EOF'
-----BEGIN CERTIFICATE-----
...
-----END CERTIFICATE-----
```
