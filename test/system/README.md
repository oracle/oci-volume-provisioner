# System Testing

Some scripts to test the OCI volume provisioner on a real Kubernetes cluster.

## Usage

We first need to setup the environment. The following must be defined:

* $DOCKER_REGISTRY_USERNAME
* DOCKER_REGISTRY_PASSWORD
* $OCICONFIG or $OCICONFIG_VAR
* $KUBECONFIG or $KUBECONFIG_VAR

Note: If set, OCICONFIG/KUBECONFIG must contain the path to the required
files. Alternatively, OCICONFIG/KUBECONFIG_VAR must contain the content
of the required files. If both are set, the former will take precedence.

We can then run the system test as follows:

```
cd test/system
./runner.py
```

