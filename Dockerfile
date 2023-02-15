FROM quay.io/skopeo/stable:v1.10.0

RUN yum -y update && \
    yum -y install jq vim && \
    yum -y clean all && \
    rm -rf /var/cache/dnf/* /var/log/dnf* /var/log/yum* && \
    mkdir -p /images

# Add docker cli
COPY --from=docker.io/library/docker:20.10.21 /usr/local/bin/docker /usr/local/bin/

# Add buildx plugin from github
ARG ARCH
RUN mkdir -p /root/.docker/cli-plugins/ && \
    curl -sLo /root/.docker/cli-plugins/docker-buildx \
        https://github.com/docker/buildx/releases/download/v0.10.0/buildx-v0.10.0.linux-${ARCH} && \
    chmod +x /root/.docker/cli-plugins/docker-buildx

WORKDIR /images
COPY build/hangar-linux-${ARCH}-* /usr/local/bin/
RUN mv /usr/local/bin/hangar-linux-${ARCH}-* /usr/local/bin/hangar
ENV SOURCE_REGISTRY="" \
    SOURCE_USERNAME="" \
    SOURCE_PASSWORD="" \
    DEST_REGISTRY="" \
    DEST_USERNAME="" \
    DEST_PASSWORD=""

ENTRYPOINT ["hangar"]
