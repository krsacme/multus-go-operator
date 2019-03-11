FROM centos

ENV MULTUS_VER=v3.1

RUN curl -o /tmp/multus.tar.gz https://github.com/intel/multus-cni/releases/download/${MULTUS_VER}/multus-cni_${MULTUS_VER}_linux_amd64.tar.gz --location && \ 
    tar -xf /tmp/multus.tar.gz --directory /tmp/ && \
    cp /tmp/multus-cni_${MULTUS_VER}_linux_amd64/multus-cni /root/multus-cni-$MULTUS_VER

CMD rm -f /opt/cni/bin/multus* && \
    cp /root/multus-cni-${MULTUS_VER} /opt/cni/bin && \
    ln -s ./multus-cni-${MULTUS_VER} /opt/cni/bin/multus && \
    cp /etc/multus-cni/70-multus-flannel.conf /etc/cni/net.d/
