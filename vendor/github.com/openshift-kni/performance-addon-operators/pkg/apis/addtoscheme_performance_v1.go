package apis

import (
	"github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, v1.SchemeBuilder.AddToScheme)
}
