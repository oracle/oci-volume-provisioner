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

package framework

import (
	"encoding/json"
	"io/ioutil"
)

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

// CanaryMetrics holds the value of the test results to form the canary json file
type CanaryMetrics struct {
	CMBlockSimple, CMBlockExt3, CMBlockNoAD, CMBlockVolumeFromBackup,
	CMFssMnt, CMFssSubnet, CMFssNoParam int32
	StartTime, EndTime string
}

// CMGlobal holds the canary metrics
var CMGlobal CanaryMetrics

// PopulateCanaryMetrics populate the canary metric to be passed as a json
func PopulateCanaryMetrics(metric string, value string) {
}

// MetricsToJSON takes the canary metrics and writes them to a file in json format.
func (f *Framework) MetricsToJSON(metricsBody CanaryMetrics) {
	cmJSON, err := json.Marshal(metricsBody)
	if err != nil {
		Logf("Invalid json format for the canary metrics")
	}
	ioutil.WriteFile(f.CheckEnvVar(MetricsFile), cmJSON, 0644)
}

// PopulateTestSuccessCanaryMetrics populated the canary metrics based on the test and result
func PopulateTestSuccessCanaryMetrics(testName string, result bool) {
	var metric, value string
	switch testName {
	case "Should be possible to create a persistent volume claim for a block storage (PVC).":
		metric = CMBlockSimple
	case "Should be possible to create a persistent volume claim (PVC) for a block storage of Ext3 file system.":
		metric = CMBlockExt3
	case "Should be possible to create a persistent volume claim (PVC) for a block storage with no AD specified.":
		metric = CMBlockNoAD
	case "Should be possible to backup a volume and restore the created backup.":
		metric = CMBlockVolumeFromBackup
	case "Should be possible to create a persistent volume claim (PVC) for a FSS with a mnt target specified.":
		metric = CMFssMnt
	case "Should be possible to create a persistent volume claim (PVC) for a FSS with a subnet id specified.":
		metric = CMFssSubnet
	case "Should be possible to create a persistent volume claim (PVC) for a FSS no mnt target or subnet id specified.":
		metric = CMFssNoParam
	}
	switch result {
	case true:
		// return 1 on failure
		value = "1"
	case false:
		// return 0 on success
		value = "0"
	}
	PopulateCanaryMetrics(metric, value)
}
