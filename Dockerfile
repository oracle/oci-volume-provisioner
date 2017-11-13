FROM oraclelinux:7.3

RUN yum install -y openssl ca-certificates

COPY dist/oci-volume-provisioner /

CMD ["/oci-volume-provisioner"]
