#!/usr/bin/env bash
if [ $# -eq 0 ]; then
    echo "Missing arguments. Use: ./scripts/generate-ocis-configmap.sh <user ocid> <fingerprint> <tennancy ocid>"
    exit 1
fi  
USER_OCID=$1
FINGERPRINT=$2
TENNANCY_OCID=$3

echo 'Generating config map'
cat > tmp.yaml << EOF
kind: ConfigMap
apiVersion: v1
metadata:
  name: oci-volume-provisioner
  namespace: oci
  labels:
    k8s-app: oci-provisioner
data:
  config.cfg: |
    [Global]
    user=$USER_OCID
    fingerprint=$FINGERPRINT
    key-file=/etc/pem/apikey.pem
    tenancy=$TENNANCY_OCID
EOF

kubectl create -f tmp.yaml
rm tmp.yaml
