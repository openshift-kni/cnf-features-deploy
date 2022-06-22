package client

import (
	"context"
	"reflect"
	"time"

	"github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// WaitForObject waits for an object of the given type and given key to appear
// on the API server.
func WaitForObject(key client.ObjectKey, object client.Object) {

	gomega.Eventually(func() error {
		return Client.Get(context.Background(), key, object)
	}, 120*time.Second, 1*time.Second).
		WithOffset(1).
		ShouldNot(
			gomega.HaveOccurred(),
			"Object [%s] not found for key [%s]", reflect.TypeOf(object), key,
		)
}
