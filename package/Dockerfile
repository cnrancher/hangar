FROM registry.suse.com/bci/bci-base:latest

RUN zypper up -y && \
    zypper in -y -f libdevmapper1_03 bash-completion vim && \
    zypper clean

# Add skopeo binary
COPY --from=docker.io/cnrancher/hardened-skopeo:v1.12.0 /usr/local/bin/skopeo /usr/local/bin/
RUN skopeo -v

WORKDIR /images
ARG ARCH
COPY build/hangar-linux-${ARCH}-* /usr/local/bin/
RUN mv /usr/local/bin/hangar-linux-${ARCH}-* /usr/local/bin/hangar
RUN echo "source <(hangar completion bash)" >> /root/.bashrc
ENV SOURCE_REGISTRY="" \
    SOURCE_USERNAME="" \
    SOURCE_PASSWORD="" \
    DEST_REGISTRY="" \
    DEST_USERNAME="" \
    DEST_PASSWORD=""

ENTRYPOINT ["hangar"]
