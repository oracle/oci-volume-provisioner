# System Testing

Some scripts to test the OCI volume provisioner on a real Kubernetes cluster.

## Usage

We first need to setup the environment. The following must be defined:

* $KUBECONFIG or $KUBECONFIG_VAR

If the --check-oci argument is going to be set, then the following will also
need to be defined:

* $OCICONFIG or $OCICONFIG_VAR

Note: If set, OCICONFIG/KUBECONFIG must contain the path to the required
files. Alternatively, OCICONFIG_VAR/KUBECONFIG_VAR must contain the content
of the required files (base64 encoded). If both are set, the former will
take precedence.

* $MNT_TARGET_OCID

Mount target ocid used to attach file systems to for the oci-fss provisioner type.
If not mount target ocid is specified, the volume provisioner will randomly select one
from the existing ones. If no mount targets exist, a new one will be created.

We can then run the system test as follows:

```
cd test/system
./runner.py
```

