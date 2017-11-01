package provisioner

import (
	"testing"

	"github.com/kubernetes-incubator/external-storage/lib/controller"
)

func TestResolveFSType(t *testing.T) {
	var fst string
	parameters := make(map[string]string)
	options := controller.VolumeOptions{Parameters: parameters}
	// test default fsType of 'ext4' is always returned.
	fst = resolveFSType(options)
	if fst == "" {
		t.Fatal("Unexpected undefined filesystem type.")
	}
	if fst != "ext4" {
		t.Fatalf("Unexpected filesystem type: '%s'.", fst)
	}
	// test configured fsType is returned.
	parameters[fsType] = "ext3"
	fst = resolveFSType(options)
	if fst == "" {
		t.Fatal("Unexpected undefined filesystem type.")
	}
	if fst != "ext3" {
		t.Fatalf("Unexpected filesystem type: '%s'.", fst)
	}
}
