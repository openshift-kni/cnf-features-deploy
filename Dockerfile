FROM golang:1.13 AS builder
WORKDIR /go/src/github.com/openshift-kni/cnf-features-deploy
COPY . .
RUN make test-bin

FROM centos:7
COPY --from=builder /go/src/github.com/openshift-kni/cnf-features-deploy/functests/functests.test /usr/bin/cnftests
CMD ["/usr/bin/cnftests"]

