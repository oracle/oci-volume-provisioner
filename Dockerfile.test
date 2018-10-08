FROM iad.ocir.io/spinnaker/oci-kube-ci:1.0.2

COPY dist /dist
COPY manifests /manifests
COPY examples /examples
COPY test /test

WORKDIR /test/system

CMD ["./runner.py"]
