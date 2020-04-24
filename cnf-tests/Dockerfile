FROM golang:1.13 AS builder
WORKDIR /go/src/github.com/openshift-kni/cnf-features-deploy
COPY . .
RUN make test-bin

FROM quay.io/openshift/origin-oc-rpms:4.5 as oc 

FROM centos:7
COPY --from=builder /go/src/github.com/openshift-kni/cnf-features-deploy/cnf-tests/bin/cnftests /usr/bin/cnftests
COPY --from=oc /go/src/github.com/openshift/oc/oc /usr/bin/oc

CMD ["/usr/bin/cnftests"]

