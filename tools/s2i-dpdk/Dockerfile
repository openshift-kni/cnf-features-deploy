# dpdk-centos7
FROM openshift/base-centos7
#FROM centos:7

LABEL maintainer="Sebastian Scheinkman <sebassch@gmail.com>"
LABEL io.openshift.s2i.scripts-url="image:///usr/libexec/s2i"
LABEL io.s2i.scripts-url="image:///usr/libexec/s2i"

ENV BUILDER_VERSION 0.1
ENV DPDK_VER 19.11
ENV DPDK_DIR /usr/src/dpdk-${DPDK_VER}
ENV RTE_TARGET=x86_64-native-linuxapp-gcc
ENV RTE_SDK=${DPDK_DIR}

LABEL io.k8s.description="Platform for building DPDK workloads" \
      io.k8s.display-name="builder 0.1" \
      io.openshift.tags="builder,dpdk"

RUN yum groupinstall -y "Development Tools"
RUN yum install -y wget \
 numactl \
 numactl-devel \
 make \
 libibverbs-devel \
 logrotate \
 rdma-core \
 ethtool \
 libpcap-devel \
 patch \
 which \
 readline-devel \
 iproute \
 libibverbs \
 lua \
 git \
 gcc && yum clean all
# Download and compile DPDK

WORKDIR /usr/src/
RUN wget http://fast.dpdk.org/rel/dpdk-${DPDK_VER}.tar.xz
RUN tar -xpvf dpdk-${DPDK_VER}.tar.xz

WORKDIR ${DPDK_DIR}

RUN sed -i -e 's/EAL_IGB_UIO=y/EAL_IGB_UIO=n/' config/common_linux
RUN sed -i -e 's/KNI_KMOD=y/KNI_KMOD=n/' config/common_linux
RUN sed -i -e 's/LIBRTE_KNI=y/LIBRTE_KNI=n/' config/common_linux
RUN sed -i -e 's/LIBRTE_PMD_KNI=y/LIBRTE_PMD_KNI=n/' config/common_linux
RUN sed -i 's/\(CONFIG_RTE_LIBRTE_MLX5_PMD=\)n/\1y/g' $DPDK_DIR/config/common_base

# Build the dpdk package with a different machine arch
RUN cp config/defconfig_x86_64-native-linuxapp-gcc config/defconfig_x86_64-hsw-linuxapp-gcc
RUN sed -i -e 's/CONFIG_RTE_MACHINE="native"/CONFIG_RTE_MACHINE="hsw"/' config/defconfig_x86_64-hsw-linuxapp-gcc

RUN make install T=${RTE_TARGET} DESTDIR=${RTE_SDK}
#
# Build TestPmd
#
WORKDIR ${DPDK_DIR}/app/test-pmd
RUN make && cp testpmd /usr/bin/testpmd

WORKDIR /usr/src/
RUN wget http://www.lua.org/ftp/lua-5.3.4.tar.gz
RUN tar xzvf lua-5.3.4.tar.gz

RUN cd ./lua-5.3.4 && make linux && make install

WORKDIR /opt/app-root/src

RUN chmod -R 777 /opt/app-root

# TODO: Copy the S2I scripts to /usr/libexec/s2i, since openshift/base-centos7 image
# sets io.openshift.s2i.scripts-url label that way, or update that label
COPY ./s2i/bin/ /usr/libexec/s2i

RUN setcap cap_sys_resource,cap_ipc_lock=+ep /usr/libexec/s2i/run

CMD ["/usr/libexec/s2i/usage"]
