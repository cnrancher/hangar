FROM registry.suse.com/bci/bci-base:latest

RUN zypper up -y && \
    zypper in -y -f libdevmapper1_03 vim && \
    zypper clean

# Add skopeo compiled binary file
COPY --from=docker.io/cnrancher/hardened-skopeo:v1.11.2 /usr/local/bin/skopeo /usr/local/bin/
RUN skopeo -v

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
