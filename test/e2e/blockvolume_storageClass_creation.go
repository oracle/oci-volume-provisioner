// Copyright 2018 Oracle and/or its affiliates. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package e2e

import (
	. "github.com/onsi/ginkgo"

	"github.com/oracle/oci-volume-provisioner/pkg/provisioner/core"
	"github.com/oracle/oci-volume-provisioner/test/e2e/framework"
)

var _ = Describe("Volume with OCI-Block Storage Class", func() {
	f := framework.NewDefaultFramework("block-basic")

	It("Should be possible to create a persistent volume claim (PVC) with storage class oci-block", func() {
		pvcJig := framework.NewPVCTestJig(f.ClientSet, "volume-provisioner-e2e-tests-pvc")
		scName := f.CreateStorageClassOrFail(framework.ClassBlock, core.ProvisionerNameDefault, nil, pvcJig.Labels)
		pvcJig.CreateAndAwaitPVCOrFail(f.Namespace.Name, framework.MinVolumeBlock, scName, nil)
	})

})
