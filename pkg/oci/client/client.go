package client

import baremetal "github.com/oracle/bmcs-go-sdk"

// FromConfig creates a baremetal client from the given configuration.
func FromConfig(cfg *Config) (*baremetal.Client, error) {
	ociClient, err := baremetal.NewClient(
		cfg.Auth.UserOCID,
		cfg.Auth.TenancyOCID,
		cfg.Auth.Fingerprint,
		baremetal.PrivateKeyBytes([]byte(cfg.Auth.PrivateKey)),
		baremetal.Region(cfg.Auth.Region))
	if err != nil {
		return nil, err
	}
	return ociClient, nil
}
