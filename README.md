# OCI Volume Provisioner

A volume provisioner that enables persistent storage when running Kubernetes on Oracle OCI.

## What is it?

Dynamic volume provisioning, a feature unique to Kubernetes, allows storage
volumes to be created on-demand. Without dynamic provisioning, cluster
administrators have to manually make calls to their cloud or storage provider
to create new storage volumes, and then create PersistentVolume objects to
represent them in Kubernetes.

The dynamic provisioning feature eliminates the need for cluster administrators
to pre-provision storage. Instead, it automatically provisions storage when it
is requested by users.

It achieves this by creating a storage class and provisioning any storage
claims that assiciated with this claim.

This is an external (out of tree) dynamic volume provisioner for OCI and Kubernetes. 
It uses the OCI Volume driver to do the actual provisioning of Storage Volumes.

## Prerequisites

+ Install the [Oracle OCI flex volume driver](https://github.com/oracle/oci-flexvolume-driver)
+ Kubernetes 1.6 + 

## Developing

Make and push the image

```
make push
```

Configure the RBAC roles: 

```
kubectl create -f manifests/auth/serviceaccount.yaml
kubectl create -f manifests/auth/clusterrole.yaml
kubectl create -f manifests/auth/clusterrolebinding.yaml
```

Configure the OCI config map, OCI secret and docker registry secret:

```
./scripts/generate-oci-configmap.sh <user ocid> <fingerprint> <tennacy ocid>
./scripts/generate-oci-secret.sh <oci_api_key.pem>
./scripts/generate-docker-registry-secret.sh <username> <password> <email>
```

Create the oci storage class:

```
kubectl create -f manifests/storage-class.yaml
```

Finally install the OCI volume provisioner:

```
kubectl create -f dist/oci-volume-provisioner.yaml
```

## Example claim yaml

The storageClassName must be "oci" and the matchlabels should contain the name
of your compartment (optional) and the availability domain (required). This is
one of {REGION}-AD-1, {REGION}-AD-2 or {REGION}-AD-3. 

```yaml
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: demooci
spec:
  storageClassName: "oci"
  selector: 
    matchLabels:
      oci-availability-domain: "PHX-AD-1"
      # optional compartment name
      oci-compartment: "bristol-cloud"
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 50Gi
```

## Example pod yaml

```yaml 
kind: Pod
apiVersion: v1
metadata:
  name: task-pv-pod
spec:
  volumes:
    - name: task-pv-storage
      persistentVolumeClaim:
      claimName: demooci
  containers:
    - name: task-pv-container
      image: nginx
      ports:
        - containerPort: 80
          name: "http-server"
      volumeMounts:
      - mountPath: "/usr/share/nginx/html"
        name: task-pv-storage
```
