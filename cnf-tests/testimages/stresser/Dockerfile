
FROM golang:1.13 AS builder
ENV PKG_NAME=github.com/openshift-kni/cnf-features-deploy
ENV PKG_PATH=/go/src/$PKG_NAME
ENV TESTER_PATH=$PKG_PATH/cnf-tests/testimages/stresser

RUN mkdir -p $PKG_PATH

COPY . $PKG_PATH/
WORKDIR $TESTER_PATH

RUN go build -mod=vendor -o /stresser

FROM centos:7
COPY --from=builder /stresser /usr/bin/stresser
ENTRYPOINT ["/usr/bin/stresser", "-logtostderr"]
