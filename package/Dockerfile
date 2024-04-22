FROM registry.suse.com/bci/bci-base:15.6

WORKDIR /hangar
COPY package/entrypoint.sh /
COPY bin/hangar /usr/local/bin/hangar
COPY package/default-policy.json /etc/containers/policy.json
COPY package/registries.d/default.yaml /etc/containers/registries.d/default.yaml

RUN zypper up -y && \
    zypper in -y -f libdevmapper1_03 bash-completion vim wget && \
    zypper clean && \
    hangar version && \
    echo "# Add hangar bash completion" >> /root/.bashrc && \
    echo "source <(hangar completion bash)" >> /root/.bashrc

ENTRYPOINT [ "/entrypoint.sh" ]
