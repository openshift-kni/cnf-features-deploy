FROM centos:8 as builder

WORKDIR /oslat
COPY . .

RUN dnf update -y && \
dnf install -y \
make \
gcc \
numactl-devel && \
dnf clean all

RUN make

FROM centos:8

RUN dnf update -y && \
dnf install -y numactl-libs && \
dnf clean all

COPY --from=builder /oslat/oslat /usr/bin/oslat
COPY run.sh /usr/local/bin/run.sh

ENTRYPOINT ["/usr/local/bin/run.sh"]
