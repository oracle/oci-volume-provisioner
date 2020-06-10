#### :warning: oci-volume-provisioner is now being maintained at https://github.com/oracle/oci-cloud-controller-manager/tree/master/pkg/volume. This repository will be archived soon.
---
# OCI Volume Provisioner

[![wercker status](https://app.wercker.com/status/0bb764451c28a60b4260d76754f02118/s/master "wercker status")](https://app.wercker.com/project/byKey/0bb764451c28a60b4260d76754f02118)
[![Go Report Card](https://goreportcard.com/badge/github.com/oracle/oci-volume-provisioner)](https://goreportcard.com/report/github.com/oracle/oci-volume-provisioner)

The OCI Volume Provisioner enables [dynamic provisioning][1] of storage resources when running Kubernetes on Oracle Cloud Infrastructure.
It uses the [OCI Flexvolume Driver][2] to bind storage resources to Kubernetes nodes. The volume provisioner offers support for

* [Block Volumes][5]

## Prerequisites

+ Install the [OCI flexvolume driver][2]
+ Kubernetes 1.6 +

## Install

The oci-volume-provisioner is provided as a Kubernetes deployment.

### Submit configuration as a Kubernetes secret

Create a config.yaml file with contents similar to the following. This file will contain authentication
information necessary to authenticate with the OCI APIs and provision block storage volumes.
The `passphrase` field can be left out if your key has no passphrase.

```yaml
auth:
  tenancy: ocid1.tenancy.oc1..aaaaaaaatyn7scrtwt...
  user: ocid1.user.oc1..aaaaaaaao235lbcxvdrrqlr...
  key: |
    -----BEGIN RSA PRIVATE KEY-----
    MIIEowIBAAKCAQEUjVBnOgC4wA3j6CeTc6hIA9B3iwuJKyR8i7w...
    -----END RSA PRIVATE KEY-----
  passphrase: supersecretpassphrase
  fingerprint: aa:bb:cc:dd:ee:ff:gg:hh:ii:jj:kk:ll:mm:nn:oo:pp
  region: us-phoenix-1

````

Submit this as a Kubernetes Secret.

```bash
kubectl create secret generic oci-volume-provisioner \
    -n kube-system \
    --from-file=config.yaml=config.yaml
```

### OCI Permissions

Please ensure that the credentials used in the secret have the following privileges in the OCI API by creating a [policy](https://docs.us-phoenix-1.oraclecloud.com/Content/Identity/Concepts/policysyntax.htm) tied to a group or user.

```
Allow group <name> to manage volumes in compartment <compartment>
Allow group <name> to manage file-systems in compartment <compartment>
```


## Deploy the OCI Volume Provisioner

First select the release to deploy. These are listed here. (https://github.com/oracle/oci-volume-provisioner/releases/latest)

If your cluster is configured to use [RBAC][3] you will need to submit the following, replacing the <VERSION> placeholder with the selected version:

```
kubectl apply -f https://github.com/oracle/oci-volume-provisioner/releases/download/<VERSION>/oci-volume-provisioner-rbac.yaml
```

Deploy the volume provisioner into your Kubernetes cluster:

```
kubectl apply -f https://github.com/oracle/oci-volume-provisioner/releases/download/<VERSION>/oci-volume-provisioner.yaml
```

Deploy the volume provisioner storage classes:

```
kubectl apply -f https://github.com/oracle/oci-volume-provisioner/releases/download/<VERSION>/storage-class.yaml
kubectl apply -f https://github.com/oracle/oci-volume-provisioner/releases/download/<VERSION>/storage-class-ext3.yaml

```

Lastly, verify that the oci-volume-provisioner is running in your cluster. By default it runs in the 'kube-system' namespace.

```
kubectl -n kube-system get po | grep oci-volume-provisioner
```

### Below is an example of deploying version '1.0.0'

```
kubectl apply -f https://github.com/oracle/oci-volume-provisioner/releases/download/1.0.0/oci-volume-provisioner.yaml
kubectl apply -f https://github.com/oracle/oci-volume-provisioner/releases/download/1.0.0/oci-volume-provisioner.yaml
kubectl apply -f https://github.com/oracle/oci-volume-provisioner/releases/download/1.0.0/storage-class.yaml
kubectl apply -f https://github.com/oracle/oci-volume-provisioner/releases/download/1.0.0/storage-class-ext3.yaml

```

## Tutorial

In this example we'll use the OCI Volume Provisioner to create persistent storage for an NGINX Pod.

### Create a PVC

Next we'll create a [PersistentVolumeClaim][4] (PVC).

The storageClassName must match the "oci" storage class supported by the provisioner.

The matchLabels should contain the (shortened) Availability Domain (AD) within
which you want to provision the volume. For example in Phoenix that might be
`PHX-AD-1`, in Ashburn `US-ASHBURN-AD-1`, in Frankfurt `EU-FRANKFURT-1-AD-1`,
and in London `UK-LONDON-1-AD-1`.

```yaml
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: nginx-volume
spec:
  storageClassName: "oci"
  selector:
    matchLabels:
      failure-domain.beta.kubernetes.io/zone: "PHX-AD-1"
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 50Gi
```

After submitting the PVC, you should see a block storage volume available in your OCI tenancy.

### Create a Kubernetes Pod that references the PVC

Now you have a PVC, you can create a Kubernetes Pod that will consume the storage.

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: nginx
spec:
  volumes:
    - name: nginx-storage
      persistentVolumeClaim:
        claimName: nginx-volume
  containers:
    - name: nginx
      image: nginx
      ports:
        - containerPort: 80
      volumeMounts:
      - mountPath: "/usr/share/nginx/html"
        name: nginx-storage
```

### Create a block volume from a backup

You can use annotations to create a volume from an existing backup. Simply use an annotation and reference the volume OCID.

```yaml
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: block-volume-from-backup
  annotations:
    volume.beta.kubernetes.io/oci-volume-source: ocid...
spec:
  storageClassName: "oci"
  selector:
    matchLabels:
      failure-domain.beta.kubernetes.io/zone: "PHX-AD-1"
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 50Gi
```

## Misc

You can add a prefix to volume display names by setting an `OCI_VOLUME_NAME_PREFIX` environment variable.

## Contributing

`oci-volume-provisioner` is an open source project. See [CONTRIBUTING](CONTRIBUTING.md) for
details.

Oracle gratefully acknowledges the contributions to this project that have been made
by the community.

## License

Copyright (c) 2017, Oracle and/or its affiliates. All rights reserved.

`oci-volume-provisioner` is licensed under the Apache License 2.0.

See [LICENSE](LICENSE) for more details.

[1]: http://blog.kubernetes.io/2016/10/dynamic-provisioning-and-storage-in-kubernetes.html
[2]: https://github.com/oracle/oci-flexvolume-driver
[3]: https://kubernetes.io/docs/admin/authorization/rbac/
[4]: https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistentvolumeclaims
[5]: https://docs.us-phoenix-1.oraclecloud.com/Content/Block/Concepts/overview.htm
