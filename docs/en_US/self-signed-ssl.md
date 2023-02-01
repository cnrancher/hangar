# Self-signed SSL Certificate

If the image registry is a private image registry (self-signed Harbor) using a self-signed SSL Certificate, please refer to the following section for configuration.

## Docker

FYI: [Use self-signed certificates](https://docs.docker.com/registry/insecure/#use-self-signed-certificates) (from step 3)

Copy the SSL public key to `/etc/docker/certs.d/<registry-url>/ca.crt`:

```console
# sudo mkdir -p /etc/docker/certs.d/${REGISTRY_URL}/
# cp public.crt /etc/docker/certs.d/${REGISTRY_URL}/ca.crt
```

## Docker Buildx

> FYI: <https://github.com/docker/buildx/issues/80#issuecomment-533844117>

This tool uses `docker-buildx` to create a Manifest List. When using self-signed SSL, you need to paste the public key into `/etc/ssl/certs/ca-certificates.crt` file.

```console
# cat >> /etc/ssl/certs/ca-certificates.crt <<'EOF'
-----BEGIN CERTIFICATE-----
...
-----END CERTIFICATE-----
EOF
```
