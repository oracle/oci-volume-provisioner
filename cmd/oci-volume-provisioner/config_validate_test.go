package main

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestValidateConfig(t *testing.T) {
	testCases := []struct {
		name string
		in   *Config
		errs field.ErrorList
	}{
		{
			name: "valid",
			in: &Config{
				Auth: AuthConfig{
					Region:      "us-phoenix-1",
					TenancyOCID: "ocid1.tenancy.oc1..aaaaaaaatyn7scrtwtqedvgrxgr2xunzeo6uanvyhzxqblctwkrpisvke4kq",
					UserOCID:    "ocid1.user.oc1..aaaaaaaai77mql2xerv7cn6wu3nhxang3y4jk56vo5bn5l5lysl34avnui3q",
					PrivateKey:  "-----BEGIN RSA PRIVATE KEY----- (etc)",
					Fingerprint: "8c:bf:17:7b:5f:e0:7d:13:75:11:d6:39:0d:e2:84:74",
				},
			},
			errs: field.ErrorList{},
		}, {
			name: "missing_region",
			in: &Config{
				Auth: AuthConfig{
					TenancyOCID: "ocid1.tenancy.oc1..aaaaaaaatyn7scrtwtqedvgrxgr2xunzeo6uanvyhzxqblctwkrpisvke4kq",
					UserOCID:    "ocid1.user.oc1..aaaaaaaai77mql2xerv7cn6wu3nhxang3y4jk56vo5bn5l5lysl34avnui3q",
					PrivateKey:  "-----BEGIN RSA PRIVATE KEY----- (etc)",
					Fingerprint: "8c:bf:17:7b:5f:e0:7d:13:75:11:d6:39:0d:e2:84:74",
				},
			},
			errs: field.ErrorList{
				&field.Error{Type: field.ErrorTypeRequired, Field: "auth.region", BadValue: ""},
			},
		}, {
			name: "missing_tenancy",
			in: &Config{
				Auth: AuthConfig{
					Region:      "us-phoenix-1",
					UserOCID:    "ocid1.user.oc1..aaaaaaaai77mql2xerv7cn6wu3nhxang3y4jk56vo5bn5l5lysl34avnui3q",
					PrivateKey:  "-----BEGIN RSA PRIVATE KEY----- (etc)",
					Fingerprint: "8c:bf:17:7b:5f:e0:7d:13:75:11:d6:39:0d:e2:84:74",
				},
			},
			errs: field.ErrorList{
				&field.Error{Type: field.ErrorTypeRequired, Field: "auth.tenancy", BadValue: ""},
			},
		}, {
			name: "missing_user",
			in: &Config{
				Auth: AuthConfig{
					Region:      "us-phoenix-1",
					TenancyOCID: "ocid1.tenancy.oc1..aaaaaaaatyn7scrtwtqedvgrxgr2xunzeo6uanvyhzxqblctwkrpisvke4kq",
					PrivateKey:  "-----BEGIN RSA PRIVATE KEY----- (etc)",
					Fingerprint: "8c:bf:17:7b:5f:e0:7d:13:75:11:d6:39:0d:e2:84:74",
				},
			},
			errs: field.ErrorList{
				&field.Error{Type: field.ErrorTypeRequired, Field: "auth.user", BadValue: ""},
			},
		}, {
			name: "missing_key",
			in: &Config{
				Auth: AuthConfig{
					Region:      "us-phoenix-1",
					TenancyOCID: "ocid1.tenancy.oc1..aaaaaaaatyn7scrtwtqedvgrxgr2xunzeo6uanvyhzxqblctwkrpisvke4kq",
					UserOCID:    "ocid1.user.oc1..aaaaaaaai77mql2xerv7cn6wu3nhxang3y4jk56vo5bn5l5lysl34avnui3q",
					Fingerprint: "8c:bf:17:7b:5f:e0:7d:13:75:11:d6:39:0d:e2:84:74",
				},
			},
			errs: field.ErrorList{
				&field.Error{Type: field.ErrorTypeRequired, Field: "auth.key", BadValue: ""},
			},
		}, {
			name: "missing_figerprint",
			in: &Config{
				Auth: AuthConfig{
					Region:      "us-phoenix-1",
					TenancyOCID: "ocid1.tenancy.oc1..aaaaaaaatyn7scrtwtqedvgrxgr2xunzeo6uanvyhzxqblctwkrpisvke4kq",
					UserOCID:    "ocid1.user.oc1..aaaaaaaai77mql2xerv7cn6wu3nhxang3y4jk56vo5bn5l5lysl34avnui3q",
					PrivateKey:  "-----BEGIN RSA PRIVATE KEY----- (etc)",
				},
			},
			errs: field.ErrorList{
				&field.Error{Type: field.ErrorTypeRequired, Field: "auth.fingerprint", BadValue: ""},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateConfig(tt.in)
			if !reflect.DeepEqual(result, tt.errs) {
				t.Errorf("ValidateConfig(%#v)\n=> %#v\nExpected: %#v", tt.in, result, tt.errs)
			}
		})
	}
}
