variable "tenancy_ocid" {
  default = "ocid1.tenancy.oc1..aaaaaaaatyn7scrtwtqedvgrxgr2xunzeo6uanvyhzxqblctwkrpisvke4kq"
}

variable "user_ocid" {
  default = "ocid1.user.oc1..aaaaaaaao235lbcxvdrrqlrpwv4qvil2xzs4544h3lof4go3wz2ett6arpeq"
}

variable "fingerprint" {
  default = "2c:29:18:b4:86:a5:d4:02:07:f4:41:6f:7d:64:02:11"
}

variable "private_key_path" {
  default = "/tmp/oci_api_key.pem"
}

variable "compartment_ocid" {
  default = "ocid1.compartment.oc1..aaaaaaaa6yrzvtwcumheirxtmbrbrya5lqkr7k7lxi34q3egeseqwlq2l5aq"
}

variable "availability_domain" {
  default = "NWuj:PHX-AD-2"
}

variable "region" {
  default = "us-phoenix-1"
}

variable "test_id" {}

provider "oci" {
  tenancy_ocid     = "${var.tenancy_ocid}"
  user_ocid        = "${var.user_ocid}"
  fingerprint      = "${var.fingerprint}"
  private_key_path = "${var.private_key_path}"
  region           = "${var.region}"
}

data "oci_identity_availability_domains" "ADs" {
  compartment_id = "${var.tenancy_ocid}"
}

resource "oci_core_volume" "test_volume" {
  availability_domain = "${var.availability_domain}"
  compartment_id      = "${var.compartment_ocid}"
  display_name        = "volume_provisioner_system_test${var.test_id}"
  size_in_gbs         = "50"
}

resource "oci_core_volume_backup" "test_volume_backup" {
  volume_id    = "${oci_core_volume.test_volume.id}"
  display_name = "backup_volume_provisioner_system_test${var.test_id}"
}

output "volume_ocid" {
  value = "${oci_core_volume.test_volume.id}"
}

output "availability_domain" {
  value = "${oci_core_volume.test_volume.availability_domain}"
}
