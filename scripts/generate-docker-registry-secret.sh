#!/bin/bash
if [ $# -eq 0 ]; then
    echo "Missing arguments. Use: ./scripts/generate-docker-registry-secret.sh <username> <password> <email address>"
    exit 1
fi  

kubectl -n kube-system create secret docker-registry odx-docker-pull-secret \
--docker-server="wcr.io" \
--docker-username="$1" \
--docker-password="$2" \
--docker-email="$3"
