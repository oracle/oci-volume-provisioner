package client

import (
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func validateAuthConfig(c AuthConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if c.Region == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("region"), ""))
	}
	if c.TenancyOCID == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("tenancy"), ""))
	}
	if c.UserOCID == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("user"), ""))
	}
	if c.PrivateKey == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("key"), ""))
	}
	if c.Fingerprint == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("fingerprint"), ""))
	}
	return allErrs
}

// ValidateConfig validates the OCI Volume Provisioner config file.
func ValidateConfig(c *Config) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validateAuthConfig(c.Auth, field.NewPath("auth"))...)
	return allErrs
}
