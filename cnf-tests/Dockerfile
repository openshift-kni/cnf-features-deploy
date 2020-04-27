FROM golang:1.13 AS builder
WORKDIR /go/src/github.com/openshift-kni/cnf-features-deploy
COPY . .
RUN make test-bin

FROM quay.io/openshift/origin-oc-rpms:4.5 as oc 

FROM centos:7


RUN mkdir -p /usr/local/etc/cnf

COPY --from=oc /go/src/github.com/openshift/oc/oc /usr/bin/oc
COPY --from=builder /go/src/github.com/openshift-kni/cnf-features-deploy/cnf-tests/bin/cnftests /usr/bin/cnftests
COPY --from=builder /go/src/github.com/openshift-kni/cnf-features-deploy/cnf-tests/bin/configsuite /usr/bin/configsuite
COPY --from=builder /go/src/github.com/openshift-kni/cnf-features-deploy/cnf-tests/bin/validationsuite /usr/bin/validationsuite
COPY --from=builder /go/src/github.com/openshift-kni/cnf-features-deploy/cnf-tests/test-run.sh /usr/bin/test-run.sh
COPY --from=builder /go/src/github.com/openshift-kni/cnf-features-deploy/cnf-tests/bin/mirror /usr/bin/mirror
COPY --from=builder /go/src/github.com/openshift-kni/cnf-features-deploy/cnf-tests/testimages/images.json /usr/local/etc/cnf
COPY --from=builder /go/src/github.com/openshift-kni/cnf-features-deploy/cnf-tests/bin/cnftests-sha.txt /usr/local/etc/cnf

ENV SUITES_PATH=/usr/bin/

CMD ["/usr/bin/test-run.sh"]
