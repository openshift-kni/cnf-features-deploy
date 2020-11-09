# This Dockerfile is the temporary solution until we will integrate oslat binary and test into cnf-tests
FROM golang:1.14 AS builder
WORKDIR /go/src/github.com/openshift-kni/performance-addon-operator
COPY . .
RUN make dist-functests
RUN git rev-list -1 HEAD > ./latency-test-sha.txt

FROM quay.io/openshift/origin-oc-rpms:4.6 AS oc

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

COPY --from=oc /go/src/github.com/openshift/oc/oc /usr/bin/oc

COPY --from=builder /go/src/github.com/openshift-kni/performance-addon-operator/functests/0_config/0_config.test /0_config.test
COPY --from=builder /go/src/github.com/openshift-kni/performance-addon-operator/functests/4_latency/4_latency.test /4_latency.test
COPY --from=builder /go/src/github.com/openshift-kni/performance-addon-operator/latency-test-sha.txt /latency-test-sha.txt
COPY --from=builder /go/src/github.com/openshift-kni/performance-addon-operator/functests/4_latency/run-tests.sh /usr/bin/run-tests.sh
