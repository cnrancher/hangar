FROM registry.suse.com/bci/bci-base:latest

WORKDIR /hangar

# Add skopeo binary
COPY --from=docker.io/cnrancher/hardened-skopeo:v1.14.1 /usr/bin/skopeo /usr/bin/

ARG ARCH
COPY build/hangar-linux-${ARCH}-* /usr/bin/
RUN zypper up -y && \
    zypper in -y -f libdevmapper1_03 bash-completion vim && \
    zypper clean && \
    mv /usr/bin/hangar-linux-${ARCH}-* /usr/bin/hangar && \
    echo "source <(hangar completion bash)" >> /root/.bashrc &&\
    skopeo -v

ENTRYPOINT ["/bin/bash"]
