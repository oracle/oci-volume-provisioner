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

FROM ubuntu:16.04

ARG TERRAFORM_VERSION=0.10.7
ARG OCI_TERRAFORM_PROVIDER_VERSION="2.0.2"
ARG KUBECTL_VERSION=v1.9.4

# Installs the required dependencies.
RUN apt-get update && apt-get install -y \
    git \
    libssl-dev \
    openssh-client \
    python \
    python-pip \
    python-yaml \
    unzip \
    wget \
    curl \
    jq \
    pwgen

# Installs terraform.
RUN wget https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip
RUN unzip terraform_${TERRAFORM_VERSION}_linux_amd64.zip
RUN mv terraform /usr/bin/

# Installs the OCI terraform provider.
RUN wget https://github.com/oracle/terraform-provider-oci/releases/download/${OCI_TERRAFORM_PROVIDER_VERSION}/linux.tar.gz
RUN tar -xzvf linux.tar.gz -C /
RUN echo "providers { oci = \"/linux_amd64/terraform-provider-oci_v${OCI_TERRAFORM_PROVIDER_VERSION}\" }" > ~/.terraformrc

# Installs the python Oracle OCI client.
RUN pip install \
    oci \
    requests[security]

# Installs the kubectl client
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl
RUN chmod +x ./kubectl
RUN mv ./kubectl /usr/local/bin/kubectl
