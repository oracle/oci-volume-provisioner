FROM ubuntu:16.10

ARG PROXY=http://emea-proxy.uk.oracle.com:80
ARG TERRAFORM_VERSION=0.10.7
ARG OCI_TERRAFORM_PROVIDER_VERSION="v2.0.1"

# Installs the required dependencies.
RUN http_proxy=$PROXY apt-get update
RUN http_proxy=$PROXY apt-get install -y git
RUN http_proxy=$PROXY apt-get install -y libssl-dev
RUN http_proxy=$PROXY apt-get install -y openssh-client
RUN http_proxy=$PROXY apt-get install -y python
RUN http_proxy=$PROXY apt-get install -y python-pip
RUN http_proxy=$PROXY apt-get install -y unzip
RUN http_proxy=$PROXY apt-get install -y wget
RUN http_proxy=$PROXY apt-get install -y curl

# Installs terraform.
RUN https_proxy=$PROXY wget https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip
RUN unzip terraform_${TERRAFORM_VERSION}_linux_amd64.zip
RUN mv terraform /usr/bin/

# Installs the OCI terraform provider.
RUN https_proxy=$PROXY wget https://github.com/oracle/terraform-provider-oci/releases/download/v2.0.1/linux.tar.gz
RUN tar -xzvf linux.tar.gz -C /
RUN echo "providers { oci = \"/linux_amd64/terraform-provider-oci_${OCI_TERRAFORM_PROVIDER_VERSION}\" }" > ~/.terraformrc

# Installs the python Oracle OCI client.
RUN https_proxy=$PROXY pip install oci
RUN https_proxy=$PROXY pip install requests[security]

# Installs the kubectl client
RUN https_proxy=$PROXY curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
RUN chmod +x ./kubectl
RUN mv ./kubectl /usr/local/bin/kubectl
