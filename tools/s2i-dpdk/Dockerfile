FROM quay.io/centos/centos:stream9

LABEL maintainer="Sebastian Scheinkman <sebassch@gmail.com>"
LABEL io.openshift.s2i.scripts-url="image:///usr/libexec/s2i"
LABEL io.s2i.scripts-url="image:///usr/libexec/s2i"

ENV BUILDER_VERSION 0.1
ENV DPDK_VER 22.11.2
ENV DPDK_DIR /usr/src/dpdk-stable-${DPDK_VER}
ENV RTE_TARGET=x86_64-native-linuxapp-gcc
ENV RTE_SDK=${DPDK_DIR}
ENV PKG_CONFIG_PATH=/usr/local/lib64/pkgconfig/

LABEL io.k8s.description="Platform for building DPDK workloads" \
      io.k8s.display-name="builder 0.1" \
      io.openshift.tags="builder,dpdk"

# The second yum install is here to overcome versions mismatch between rpms
RUN yum install -y wget python3.11 python3.11-pip \
 numactl \
 numactl-devel \
 make \
 logrotate \
 ethtool \
 patch \
 which \
 readline-devel \
 iproute \
 libibverbs \
 lua \
 git \
 gcc \
 pciutils \
 xz \
 libibverbs-devel \
 expect && \
 yum clean all

RUN pip3.11 install meson ninja pyelftools

RUN cd /usr/src/ && wget http://fast.dpdk.org/rel/dpdk-${DPDK_VER}.tar.xz && tar -xpvf dpdk-${DPDK_VER}.tar.xz && rm dpdk-${DPDK_VER}.tar.xz && \
    cd dpdk-stable-${DPDK_VER} && \
    meson setup build && \
    cd build && \
    meson configure -Denable_docs=false \
                    -Dplatform=generic \
                    -Dmax_ethports=32 \
                    -Dmax_numa_nodes=8 \
                    -Dtests=false && \
    ninja && \
    ninja install && \
    echo "/usr/local/lib64" > /etc/ld.so.conf.d/dpdk.conf && \
    ldconfig

RUN ln -s ${DPDK_DIR}/build/app/dpdk-testpmd /usr/bin/testpmd

RUN mkdir -p /opt/app-root/src

WORKDIR /opt/app-root/src

RUN chmod -R 777 /opt/app-root

# TODO: Copy the S2I scripts to /usr/libexec/s2i, since openshift/base-centos7 image
# sets io.openshift.s2i.scripts-url label that way, or update that label
COPY ./s2i/bin/ /usr/libexec/s2i

RUN setcap cap_sys_resource,cap_ipc_lock=+ep /usr/libexec/s2i/run

# Allows non-root users to use dpdk-testpmd.
RUN setcap cap_ipc_lock,cap_net_raw+ep /usr/local/bin/dpdk-testpmd
RUN setcap cap_ipc_lock,cap_net_raw+ep /usr/local/bin/dpdk-test-bbdev

# Add supplementary group 801 to user 1001 in order to use the VFIO device in a non-privileged pod.
RUN groupadd -g 801 hugetlbfs
RUN useradd -u 1001 dpdk-user
RUN usermod -aG hugetlbfs dpdk-user

# This is needed for the s2i to work
# in the pod yaml we still use the runAsUser:0 we w/a the ulimit issue
USER dpdk-user

CMD ["/usr/libexec/s2i/usage"]
