FROM registry.suse.com/bci/bci-base:15.5

WORKDIR /hangar
COPY bin/hangar /usr/local/bin/hangar

RUN zypper up -y && \
    zypper in -y -f libdevmapper1_03 bash-completion vim wget && \
    zypper clean && \
    mkdir -p /etc/containers/ && \
    hangar version && \
    echo "# Add hangar bash completion" >> /root/.bashrc && \
    echo "source <(hangar completion bash)" >> /root/.bashrc

COPY package/default-policy.json /etc/containers/policy.json

ENTRYPOINT [ "/bin/bash" ]
