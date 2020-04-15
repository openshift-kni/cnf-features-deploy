package utils

import (
	. "github.com/onsi/ginkgo"
)

func BeforeAll(fn func()) {
	first := true
	BeforeEach(func() {
		if first {
			fn()
			first = false
		}
	})
}
