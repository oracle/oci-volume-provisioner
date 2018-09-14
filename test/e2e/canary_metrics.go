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

const (
	CMBlockSimple           = "volume_provisioner_block_simple"
	CMBlockExt3             = "volume_provisioner_block_ext3"
	CMBlockNoAD             = "volume_provisioner_block_no_ad"
	CMBlockVolumeFromBackup = "volume_provisioner_block_volume_from_backup"
	CMFssMnt                = "volume_provisioner_fss_mnt"
	CMFssSubnet             = "volume_provisioner_fss_subnet"
	CMFssNoParam            = "volume_provisioner_no_param"
	StartTime               = "start_time"
	EndTime                 = "end_time"
	MetricsFile             = "METRICS_FILE"
)

type CanaryMetrics struct {
	testName string
	result   int
}
