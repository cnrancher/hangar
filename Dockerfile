FROM registry.suse.com/bci/bci-base:latest

RUN zypper up -y && \
    zypper in -y -f libdevmapper1_03 vim && \
    zypper clean

# Add docker cli
COPY --from=docker.io/library/docker:23.0 /usr/local/bin/docker /usr/local/bin/
# Add docker-buildx
COPY --from=docker.io/docker/buildx-bin:latest /buildx /usr/libexec/docker/cli-plugins/docker-buildx
# Add skopeo compiled binary file
COPY --from=docker.io/cnrancher/hardened-skopeo:v1.10.0 /usr/local/bin/skopeo /usr/local/bin/

# Check docker, docker-buildx, skopeo version
RUN docker --version && \
    docker buildx version && \
    skopeo -v

WORKDIR /images
# Add buildx plugin from github
ARG ARCH
COPY build/hangar-linux-${ARCH}-* /usr/local/bin/
RUN mv /usr/local/bin/hangar-linux-${ARCH}-* /usr/local/bin/hangar
ENV SOURCE_REGISTRY="" \
    SOURCE_USERNAME="" \
    SOURCE_PASSWORD="" \
    DEST_REGISTRY="" \
    DEST_USERNAME="" \
    DEST_PASSWORD=""

ENTRYPOINT ["hangar"]
