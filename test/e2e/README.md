# E2E Tests

These tests are adapted from the [Service test suite][1] in the Kubernetes core
E2E tests.

## Running

```bash
ginkgo -v -progress test/e2e -- --kubeconfig=${HOME}/.kube/config --delete-namespace=false
```

NOTE: Test suite will fail if executed behind a `$HTTP_PROXY` that returns a
200 OK response upon failure to connect.

## Additional options

|Flag | Usage | Value |
|---|---|---|
|kubeconfig|Path to Kubeconfig file with authorization and master location information.| string|
|ociconfig|Path to OCIconfig file with cloud provider config.|string|
|namespace| Name of an existing Namespace to run tests in.| string|
|delete-namespace|If true tests will delete namespace after completion. It is only designed to make debugging easier, DO NOT turn it off by default.| bool|
|delete-namespace-on-failure|If true tests will delete their associated namespace upon completion whether the test has failed.|bool|
|mnt-target-id| Identifies the mount target id for a FSS.|string|
|subnet-id| Identifies a subnet to look for a mount target, such that a FSS can be mounted.|string|
|ad| Identifies the availability domain in which the PD resides|string|

[1]: https://github.com/kubernetes/kubernetes/blob/0cb15453dae92d8be66cf42e6c1b04e21a2d0fb6/test/e2e/network/service.go
