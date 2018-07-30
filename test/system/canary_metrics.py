#!/usr/bin/env python

# Copyright (c) 2018 Oracle and/or its affiliates. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import utils
import os
import json
import datetime
from collections import MutableMapping

class CanaryMetrics(object):

    CM_SIMPLE = "volume_provisioner_simple"
    CM_EXT3 = "volume_provisioner_ext3"
    CM_NO_AD = "volume_provisioner_no_ad"
    CM_VOLUME_FROM_BACKUP = "volume_provisioner_volume_from_backup"
    START_TIME = "start_time"
    END_TIME = "end_time"

    def __init__(self, metrics_file=None, *args, **kwargs):
        self._canaryMetrics = dict(*args, **kwargs)
        self._metrics_file = metrics_file
        self._canaryMetrics[self.START_TIME] = self.canary_metric_date()

    @staticmethod
    def canary_metric_date():
        return datetime.datetime.today().strftime('%Y-%m-%d-%H%m%S')

    def update_canary_metric(self, name, result):
        self._canaryMetrics[name] = result

    def finish_canary_metrics(self):
        self.update_canary_metric(self.END_TIME, self.canary_metric_date())
        if self._metrics_file:
            with open(self._metrics_file, 'w') as metrics_file:
                json.dump(self._canaryMetrics, metrics_file, sort_keys=True, indent=4)
