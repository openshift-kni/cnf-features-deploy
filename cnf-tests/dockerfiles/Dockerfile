FROM golang:1.13 AS builder
WORKDIR /go/src/github.com/openshift-kni/cnf-features-deploy
COPY . .
RUN make test-bin
RUN git rev-list -1 HEAD > ./cnf-tests/bin/cnftests-sha.txt

FROM quay.io/openshift/origin-oc-rpms:4.8 AS oc

FROM golang:1.13 AS builder-stresser
ENV PKG_NAME=github.com/openshift-kni/cnf-features-deploy
ENV PKG_PATH=/go/src/$PKG_NAME
ENV TESTER_PATH=$PKG_PATH/cnf-tests/testbinaries/stresser

RUN mkdir -p $PKG_PATH

COPY . $PKG_PATH/
WORKDIR $TESTER_PATH

RUN go build -mod=vendor -o /stresser

FROM golang:1.13 AS builder-sctptester
ENV PKG_NAME=github.com/openshift-kni/cnf-features-deploy
ENV PKG_PATH=/go/src/$PKG_NAME
ENV TESTER_PATH=$PKG_PATH/cnf-tests/testbinaries/sctptester

RUN mkdir -p $PKG_PATH

COPY . $PKG_PATH/
WORKDIR $TESTER_PATH

RUN go build -mod=vendor -o /sctptest

# Build oslat-runner binary
FROM golang:1.13 AS builder-oslat-runner

ENV PKG_NAME=github.com/openshift-kni/cnf-features-deploy
ENV PKG_PATH=/go/src/$PKG_NAME
ENV TESTER_PATH=$PKG_PATH/cnf-tests/testbinaries/oslat-runner

RUN mkdir -p $PKG_PATH

COPY . $PKG_PATH/
WORKDIR $TESTER_PATH

RUN go build -mod=vendor -o /oslat-runner

# Build oslat binary
FROM centos:7 as builder-oslat

ENV RT_TESTS_URL=https://git.kernel.org/pub/scm/utils/rt-tests/rt-tests.git/snapshot
ENV RT_TESTS_PKG=rt-tests-1.9

RUN yum install -y numactl-devel make gcc && \
    curl -O $RT_TESTS_URL/$RT_TESTS_PKG.tar.gz && \
    tar -xvf $RT_TESTS_PKG.tar.gz && \
    cd $RT_TESTS_PKG && \
    make oslat && \
    cp oslat /oslat

FROM centos:7

RUN yum install -y lksctp-tools iproute libhugetlbfs-utils libhugetlbfs tmux ethtool ping numactl-libs

RUN mkdir -p /usr/local/etc/cnf

COPY --from=builder-stresser /stresser /usr/bin/stresser
COPY --from=builder-sctptester /sctptest /usr/bin/sctptest

COPY --from=builder-oslat-runner /oslat-runner /usr/bin/oslat-runner
COPY --from=builder-oslat /oslat /usr/bin/oslat

COPY --from=oc /go/src/github.com/openshift/oc/oc /usr/bin/oc
COPY --from=builder /go/src/github.com/openshift-kni/cnf-features-deploy/cnf-tests/bin/cnftests /usr/bin/cnftests
COPY --from=builder /go/src/github.com/openshift-kni/cnf-features-deploy/cnf-tests/bin/configsuite /usr/bin/configsuite
COPY --from=builder /go/src/github.com/openshift-kni/cnf-features-deploy/cnf-tests/bin/validationsuite /usr/bin/validationsuite
COPY --from=builder /go/src/github.com/openshift-kni/cnf-features-deploy/cnf-tests/test-run.sh /usr/bin/test-run.sh
COPY --from=builder /go/src/github.com/openshift-kni/cnf-features-deploy/cnf-tests/bin/mirror /usr/bin/mirror
COPY --from=builder /go/src/github.com/openshift-kni/cnf-features-deploy/cnf-tests/images.json /usr/local/etc/cnf
COPY --from=builder /go/src/github.com/openshift-kni/cnf-features-deploy/cnf-tests/bin/cnftests-sha.txt /usr/local/etc/cnf

ENV SUITES_PATH=/usr/bin/

CMD ["/usr/bin/test-run.sh"]
