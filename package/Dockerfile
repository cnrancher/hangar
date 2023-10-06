FROM registry.suse.com/bci/bci-base:15.5

RUN zypper up -y && \
    zypper in -y -f libdevmapper1_03 bash-completion vim && \
    zypper clean

# Add skopeo binary
COPY --from=docker.io/cnrancher/hardened-skopeo:v1.12.0 /usr/local/bin/skopeo /usr/local/bin/

WORKDIR /hangar
ARG ARCH
COPY build/hangar-linux-${ARCH} /usr/local/bin/hangar
COPY package/entrypoint.sh /usr/local/bin/entrypoint.sh

RUN echo "# Add hangar bash completion" >> /root/.bashrc && \
    echo "source <(hangar completion bash)" >> /root/.bashrc

ENTRYPOINT [ "/bin/bash" ]
