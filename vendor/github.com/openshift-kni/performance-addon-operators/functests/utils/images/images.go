package images

import (
	"fmt"
	"os"

	. "github.com/onsi/gomega"
)

var registry string
var images map[string]imageLocation

const (
	// TestUtils is the image name to be used to retrieve the test utils image
	TestUtils = "testutils"
	// Stresser is the image name to be used to retrieve the stresser image
	Stresser = "stresser"
)

func init() {
	registry = os.Getenv("IMAGE_REGISTRY")

	images = map[string]imageLocation{
		TestUtils: {
			name:    "cnftest-utils",
			registy: "quay.io/openshift-kni/",
			version: "4.5",
		},
		Stresser: {
			name:    "stresser",
			registy: "quay.io/openshift-kni/",
			version: "4.5",
		},
	}
}

type imageLocation struct {
	name    string
	registy string
	version string
}

// For returns the image to be used for the given key
func For(name string) string {
	img, ok := images[name]
	Expect(ok).To(BeTrue(), "Image not found")

	if registry != "" {
		return fmt.Sprintf("%s%s:%s", registry, img.name, img.version)
	}
	return fmt.Sprintf("%s%s:%s", img.registy, img.name, img.version)
}
