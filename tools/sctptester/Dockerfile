
FROM golang:1.13 AS builder
WORKDIR /gotest
COPY . .
RUN go mod init github.com/openshift-kni/cnf-features-deploy/sctptester
RUN go get
RUN go build -o sctptest

FROM centos:7
COPY --from=builder /gotest/sctptest /usr/bin/sctptest
RUN yum install -y lksctp-tools
CMD ["/usr/bin/sctptest"]
