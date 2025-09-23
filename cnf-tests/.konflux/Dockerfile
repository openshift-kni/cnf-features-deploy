FROM registry.redhat.io/openshift4/ose-cli-rhel9:v4.19 AS oc

FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_9_golang_1.23 AS builder-stresser
ENV PKG_NAME=github.com/openshift-kni/cnf-features-deploy
ENV PKG_PATH=/go/src/$PKG_NAME
ENV TESTER_PATH=$PKG_PATH/cnf-tests/pod-utils/stresser
RUN mkdir -p $PKG_PATH
COPY . $PKG_PATH/
WORKDIR $TESTER_PATH
RUN go build -mod=vendor -o /stresser

FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_9_golang_1.23 AS builder-sctptester
ENV PKG_NAME=github.com/openshift-kni/cnf-features-deploy
ENV PKG_PATH=/go/src/$PKG_NAME
ENV TESTER_PATH=$PKG_PATH/cnf-tests/pod-utils/sctptester
RUN mkdir -p $PKG_PATH
COPY . $PKG_PATH/
WORKDIR $TESTER_PATH
RUN go build -mod=vendor -o /sctptest

FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_9_golang_1.23 AS builder-hugepages-allocator
ENV PKG_NAME=github.com/openshift-kni/cnf-features-deploy
ENV PKG_PATH=/go/src/$PKG_NAME
ENV TESTER_PATH=$PKG_PATH/cnf-tests/pod-utils/hugepages-allocator
RUN mkdir -p $PKG_PATH
COPY . $PKG_PATH/
WORKDIR $TESTER_PATH
RUN go build -mod=vendor -o /hugepages-allocator

FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_9_golang_1.23 AS builder-latency-test-runners
ENV PKG_NAME=github.com/openshift-kni/cnf-features-deploy
ENV PKG_PATH=/go/src/$PKG_NAME
ENV TESTER_PATH=$PKG_PATH/cnf-tests/pod-utils
RUN mkdir -p $PKG_PATH
COPY . $PKG_PATH/
WORKDIR $TESTER_PATH
RUN go build -mod=vendor -o /oslat-runner oslat-runner/main.go && \
    go build -mod=vendor -o /cyclictest-runner cyclictest-runner/main.go && \
    go build -mod=vendor -o /hwlatdetect-runner hwlatdetect-runner/main.go

FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_9_golang_1.23 AS gobuilder
WORKDIR /app
COPY . .
RUN make test-bin

# Container runtime image
FROM registry.access.redhat.com/ubi9/ubi-minimal:9.6@sha256:2f06ae0e6d3d9c4f610d32c480338eef474867f435d8d28625f2985e8acde6e8
#

RUN mkdir -p /usr/local/etc/cnf
RUN microdnf install -y lksctp-tools iproute \
      ethtool iputils procps-ng numactl-libs iptables \
      kmod realtime-tests linuxptp iperf3 nc findutils \
      python3 # python3 is needed for hwlatdetect

COPY --from=oc /usr/bin/oc /usr/bin/oc
COPY --from=gobuilder /app/cnf-tests/submodules/cluster-node-tuning-operator/build/_output/bin/latency-e2e.test /usr/bin/latency-e2e.test
COPY --from=gobuilder /app/cnf-tests/entrypoint/test-run.sh /usr/bin/test-run.sh
COPY --from=gobuilder /app/cnf-tests/bin/mirror /usr/bin/mirror
COPY --from=gobuilder /app/cnf-tests/mirror/images.json /usr/local/etc/cnf
COPY --from=builder-latency-test-runners /oslat-runner /usr/bin/oslat-runner
COPY --from=builder-latency-test-runners /cyclictest-runner /usr/bin/cyclictest-runner
COPY --from=builder-latency-test-runners /hwlatdetect-runner /usr/bin/hwlatdetect-runner
COPY --from=builder-stresser /stresser /usr/bin/stresser
COPY --from=builder-sctptester /sctptest /usr/bin/sctptest
COPY --from=builder-hugepages-allocator /hugepages-allocator /usr/bin/hugepages-allocator

RUN sed -i 's/quay.io\/openshift-kni\//registry.redhat.io\/openshift4\//g' /usr/local/etc/cnf/images.json
RUN sed -i 's/cnf-tests:4.20/cnf-tests-rhel9:v4.20/g' /usr/local/etc/cnf/images.json
RUN sed -i 's/dpdk:4.20/dpdk-base-rhel9:v4.20/g' /usr/local/etc/cnf/images.json

ENV OCP_VERSION=4.20
ENV IMAGE_REGISTRY=registry.redhat.io/openshift4/
ENV CNF_TESTS_IMAGE=cnf-tests-rhel9:v${OCP_VERSION}
ENV DPDK_TESTS_IMAGE=dpdk-base-rhel9:v${OCP_VERSION}

CMD ["/usr/bin/test-run.sh"]

LABEL com.redhat.component="cnf-tests-container" \
      name="openshift4/cnf-tests" \
      summary="Cluster verification tests image" \
      io.openshift.expose-services="" \
      io.openshift.tags="data,images" \
      io.k8s.display-name="cnf-tests" \
      io.openshift.maintainer.component="Telco CNF tests image" \
      io.openshift.maintainer.product="OpenShift Container Platform" \
      io.k8s.description="Cluster verification tests image" \
      maintainer="cnf-devel@redhat.com" \
      description="Cluster verification tests image"
