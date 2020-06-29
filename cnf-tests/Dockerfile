FROM golang:1.13 AS builder
WORKDIR /go/src/github.com/rgoulding/cnf-features-deploy
COPY . .
RUN make test-bin
RUN git rev-list -1 HEAD > ./cnf-tests/bin/cnftests-sha.txt

FROM quay.io/openshift/origin-oc-rpms:4.5 AS oc

FROM golang:1.13 AS builder-stresser
ENV PKG_NAME=github.com/rgoulding/cnf-features-deploy
ENV PKG_PATH=/go/src/$PKG_NAME
ENV TESTER_PATH=$PKG_PATH/cnf-tests/testbinaries/stresser

RUN mkdir -p $PKG_PATH

COPY . $PKG_PATH/
WORKDIR $TESTER_PATH

RUN go build -mod=vendor -o /stresser

FROM golang:1.13 AS builder-sctptester
ENV PKG_NAME=github.com/rgoulding/cnf-features-deploy
ENV PKG_PATH=/go/src/$PKG_NAME
ENV TESTER_PATH=$PKG_PATH/cnf-tests/testbinaries/sctptester

RUN mkdir -p $PKG_PATH

COPY . $PKG_PATH/
WORKDIR $TESTER_PATH

RUN go build -mod=vendor -o /sctptest

FROM centos:7

RUN yum install -y lksctp-tools iproute libhugetlbfs-utils libhugetlbfs tmux ethtool ping

RUN mkdir -p /usr/local/etc/cnf

COPY --from=builder-stresser /stresser /usr/bin/stresser
COPY --from=builder-sctptester /sctptest /usr/bin/sctptest

COPY --from=oc /go/src/github.com/openshift/oc/oc /usr/bin/oc
COPY --from=builder /go/src/github.com/rgoulding/cnf-features-deploy/cnf-tests/bin/cnftests /usr/bin/cnftests
COPY --from=builder /go/src/github.com/rgoulding/cnf-features-deploy/cnf-tests/bin/configsuite /usr/bin/configsuite
COPY --from=builder /go/src/github.com/rgoulding/cnf-features-deploy/cnf-tests/bin/validationsuite /usr/bin/validationsuite
COPY --from=builder /go/src/github.com/rgoulding/cnf-features-deploy/cnf-tests/test-run.sh /usr/bin/test-run.sh
COPY --from=builder /go/src/github.com/rgoulding/cnf-features-deploy/cnf-tests/bin/mirror /usr/bin/mirror
COPY --from=builder /go/src/github.com/rgoulding/cnf-features-deploy/cnf-tests/images.json /usr/local/etc/cnf
COPY --from=builder /go/src/github.com/rgoulding/cnf-features-deploy/cnf-tests/bin/cnftests-sha.txt /usr/local/etc/cnf

ENV SUITES_PATH=/usr/bin/

CMD ["/usr/bin/test-run.sh"]
