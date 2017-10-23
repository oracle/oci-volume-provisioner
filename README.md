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

## Building

Make and push the image

```
make push
```

Note: We publish the `oci-volume-provisioner` to a private Docker registry. You
will need a [Docker registry secret][2] to push images to it.

```bash
$ kubectl -n kube-system create secret docker-registry wcr-docker-pull-secret \
    --docker-server="registry.oracledx.com" \
    --docker-username="agent" \
    --docker-password="$DOCKER_REGISTRY_PASSWORD" \
    --docker-email="k8s@oracle.com"
```

## Configuration

An example configuration file can be found [here][1]. Download this file and
populate it with values specific to your chosen OCI identity and tenancy.
Then create the Kubernetes secret with the following command:
vim c
```bash
$ kubectl create secret generic oci-volume-provisioner \
     -n kube-system \
     --from-file=config.yaml=oci-volume-provisioner-config.yaml
```

Create the `oci` storage class:

```
kubectl create -f manifests/storage-class.yaml
```

## Deployment

Lastly deploy the volume provisioner and associated RBAC rules if your cluster is configured to use RBAC:

```
kubectl create -f dist/oci-volume-provisioner.yaml
kubectl create -f manifests/oci-volume-provisioner-rbac.yaml
```

## Usage

### Example claim yaml

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
      oci-compartment: "kubernetes-test"
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 50Gi
```

### Example pod yaml

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


[1]: https://github.com/oracle/oci-volume-provisioner/tree/master/manifests/oci-volume-provisioner-config-example.yaml
[2]: https://kubernetes.io/docs/concepts/containers/images/#creating-a-secret-with-a-docker-config
