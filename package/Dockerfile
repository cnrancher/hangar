FROM registry.suse.com/bci/bci-base:15.5

WORKDIR /hangar
COPY bin/hangar /usr/local/bin/hangar
COPY package/default-policy.json /etc/containers/policy.json
COPY package/registries.d/default.conf /etc/registries.d/default.conf

RUN zypper up -y && \
    zypper in -y -f libdevmapper1_03 bash-completion vim wget && \
    zypper clean && \
    mkdir -p /etc/containers/ && \
    hangar version && \
    echo "# Add hangar bash completion" >> /root/.bashrc && \
    echo "source <(hangar completion bash)" >> /root/.bashrc

ENTRYPOINT [ "/bin/bash" ]
