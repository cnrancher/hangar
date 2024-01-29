FROM registry.suse.com/bci/bci-base:latest

WORKDIR /images

# Add skopeo binary
COPY --from=docker.io/cnrancher/hardened-skopeo:v1.14.1 /usr/local/bin/skopeo /usr/local/bin/
COPY build/hangar-linux-${ARCH}-* /usr/local/bin/

ARG ARCH
RUN skopeo -v && \
    RUN zypper up -y && \
    zypper in -y -f libdevmapper1_03 bash-completion vim && \
    zypper clean && \
    mv /usr/local/bin/hangar-linux-${ARCH}-* /usr/local/bin/hangar && \
    echo "source <(hangar completion bash)" >> /root/.bashrc

ENTRYPOINT ["hangar"]
