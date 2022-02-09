FROM registry.ci.openshift.org/openshift/release:golang-1.17 AS builder
WORKDIR /go/src/github.com/openshift-kni/cnf-features-deploy
COPY . .
RUN make test-bin
RUN git rev-list -1 HEAD > ./cnf-tests/bin/cnftests-sha.txt

FROM quay.io/openshift/origin-oc-rpms:4.8 AS oc

FROM registry.ci.openshift.org/openshift/release:golang-1.17 AS builder-stresser
ENV PKG_NAME=github.com/openshift-kni/cnf-features-deploy
ENV PKG_PATH=/go/src/$PKG_NAME
ENV TESTER_PATH=$PKG_PATH/cnf-tests/pod-utils/stresser

RUN mkdir -p $PKG_PATH

COPY . $PKG_PATH/
WORKDIR $TESTER_PATH

RUN go build -mod=vendor -o /stresser

FROM registry.ci.openshift.org/openshift/release:golang-1.17 AS builder-sctptester
ENV PKG_NAME=github.com/openshift-kni/cnf-features-deploy
ENV PKG_PATH=/go/src/$PKG_NAME
ENV TESTER_PATH=$PKG_PATH/cnf-tests/pod-utils/sctptester

RUN mkdir -p $PKG_PATH

COPY . $PKG_PATH/
WORKDIR $TESTER_PATH

RUN go build -mod=vendor -o /sctptest

# build latency-test's runner binaries
FROM registry.ci.openshift.org/openshift/release:golang-1.17 AS builder-latency-test-runners

ENV PKG_NAME=github.com/openshift-kni/cnf-features-deploy
ENV PKG_PATH=/go/src/$PKG_NAME
ENV TESTER_PATH=$PKG_PATH/cnf-tests/pod-utils

RUN mkdir -p $PKG_PATH

COPY . $PKG_PATH/
WORKDIR $TESTER_PATH

RUN go build -mod=vendor -o /oslat-runner oslat-runner/main.go && \
    go build -mod=vendor -o /cyclictest-runner cyclictest-runner/main.go && \
    go build -mod=vendor -o /hwlatdetect-runner hwlatdetect-runner/main.go

# Build latency-test binaries
FROM centos:7 as builder-latency-test-tools

ENV RT_TESTS_URL=https://git.kernel.org/pub/scm/utils/rt-tests/rt-tests.git/snapshot
ENV RT_TESTS_PKG=rt-tests-2.0

RUN yum install -y numactl-devel make gcc && \
    curl -O $RT_TESTS_URL/$RT_TESTS_PKG.tar.gz && \
    tar -xvf $RT_TESTS_PKG.tar.gz && \
    cd $RT_TESTS_PKG && \
    make oslat hwlatdetect cyclictest && \
    cp oslat /oslat && \
    cp hwlatdetect /hwlatdetect && \
    cp cyclictest /cyclictest

FROM centos:7

# python3 is needed for hwlatdetect
RUN yum install -y lksctp-tools iproute libhugetlbfs-utils libhugetlbfs tmux ethtool ping numactl-libs linuxptp iperf3 python3

RUN mkdir -p /usr/local/etc/cnf

COPY --from=builder-stresser /stresser /usr/bin/stresser
COPY --from=builder-sctptester /sctptest /usr/bin/sctptest

COPY --from=builder-latency-test-runners /oslat-runner /usr/bin/oslat-runner
COPY --from=builder-latency-test-tools /oslat /usr/bin/oslat

COPY --from=builder-latency-test-runners /cyclictest-runner /usr/bin/cyclictest-runner
COPY --from=builder-latency-test-tools /cyclictest /usr/bin/cyclictest

COPY --from=builder-latency-test-runners /hwlatdetect-runner /usr/bin/hwlatdetect-runner
COPY --from=builder-latency-test-tools /hwlatdetect /usr/bin/hwlatdetect

COPY --from=oc /go/src/github.com/openshift/oc/oc /usr/bin/oc
COPY --from=builder /go/src/github.com/openshift-kni/cnf-features-deploy/cnf-tests/bin/cnftests /usr/bin/cnftests
COPY --from=builder /go/src/github.com/openshift-kni/cnf-features-deploy/cnf-tests/bin/configsuite /usr/bin/configsuite
COPY --from=builder /go/src/github.com/openshift-kni/cnf-features-deploy/cnf-tests/bin/validationsuite /usr/bin/validationsuite
COPY --from=builder /go/src/github.com/openshift-kni/cnf-features-deploy/cnf-tests/entrypoint/test-run.sh /usr/bin/test-run.sh
COPY --from=builder /go/src/github.com/openshift-kni/cnf-features-deploy/cnf-tests/bin/mirror /usr/bin/mirror
COPY --from=builder /go/src/github.com/openshift-kni/cnf-features-deploy/cnf-tests/mirror/images.json /usr/local/etc/cnf
COPY --from=builder /go/src/github.com/openshift-kni/cnf-features-deploy/cnf-tests/bin/cnftests-sha.txt /usr/local/etc/cnf
COPY tools/oot-driver/ /usr/src/oot-driver/

ENV SUITES_PATH=/usr/bin/

CMD ["/usr/bin/test-run.sh"]
