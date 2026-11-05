package htmx_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestHTMX runs the htlmx package's Ginkgo suite. It lives in the external
// test package on purpose: the handlers are exercised through their public
// surface, the same way consumers use them.
func TestHTMX(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HTMX Suite")
}
