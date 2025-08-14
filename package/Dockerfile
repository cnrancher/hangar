FROM registry.suse.com/bci/bci-base:15.7

RUN zypper up -y && \
    zypper in -y -f libdevmapper1_03 bash-completion vim wget && \
    zypper clean -a && \
    rm -rf /tmp/* /var/tmp/* /usr/share/doc/packages/* && \
    echo "# Add hangar bash completion" >> /root/.bashrc && \
    echo "source <(hangar completion bash)" >> /root/.bashrc

ARG TARGETARCH
WORKDIR /hangar
COPY package/entrypoint.sh /
COPY dist/hangar_linux_${TARGETARCH}*/ /usr/local/bin/
COPY package/default-policy.json /etc/containers/policy.json
COPY package/registries.d/default.yaml /etc/containers/registries.d/default.yaml

ENTRYPOINT [ "/entrypoint.sh" ]
