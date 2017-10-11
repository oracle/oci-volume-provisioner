FROM oraclelinux:7.3
COPY dist/oci-volume-provisioner /
CMD ["/oci-volume-provisioner"]
