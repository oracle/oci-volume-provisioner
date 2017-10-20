#!/usr/bin/env bash
if [ $# -eq 0 ]; then
    echo "Missing arguments. Use: ./scripts/generate-ocis-secret.sh <ocis_api_key.pem>"
    exit 1
fi

FILE=$1
if [ ! -f $FILE ]; then
    echo "Expected to find a ocis_api_key at $FILE"
    exit 1
fi

echo 'Generating secret from API key'
cat > tmp.yaml << EOF
apiVersion: v1
kind: Secret
metadata:
  name: ocisapikey
  namespace: kube-system
type: Opaque
data:
  apikey.pem: $(openssl enc -A -base64 -in $FILE)
EOF
  
kubectl create -f tmp.yaml
rm tmp.yaml
