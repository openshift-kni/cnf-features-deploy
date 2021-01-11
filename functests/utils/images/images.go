package images

import (
	"fmt"
	"os"

	gomega "github.com/onsi/gomega"
)

var (
	registry      string
	cnfTestsImage string
	dpdkTestImage string
	images        map[string]imageLocation
)

const (
	// TestUtils is the image name to be used to retrieve the test utils image
	TestUtils = "testutils"
	// Dpdk is the image name to be used to retrieve the dpdk image
	Dpdk = "dpdk"
)

func init() {
	registry = os.Getenv("IMAGE_REGISTRY")
	if registry == "" {
		registry = "quay.io/openshift-kni/"
	}

	cnfTestsImage = os.Getenv("CNF_TESTS_IMAGE")
	if cnfTestsImage == "" {
		cnfTestsImage = "cnf-tests:4.8"
	}

	dpdkTestImage = os.Getenv("DPDK_TESTS_IMAGE")
	if dpdkTestImage == "" {
		dpdkTestImage = "dpdk:4.8"
	}

	images = map[string]imageLocation{
		TestUtils: {
			registry: registry,
			image:    cnfTestsImage,
		},
		Dpdk: {
			registry: registry,
			image:    dpdkTestImage,
		},
	}
}

type imageLocation struct {
	registry string
	image    string
}

// For returns the image to be used for the given key
func For(name string) string {
	img, ok := images[name]
	gomega.Expect(ok).To(gomega.BeTrue(), "Image not found")

	return fmt.Sprintf("%s%s", img.registry, img.image)
}
