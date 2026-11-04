package httperr_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestHTTPErr runs the httperr package's Ginkgo suite. It lives in the external
// test package: the handlers and mappers are exercised through their public
// surface, the way consumers use them.
func TestHTTPErr(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "httperr Suite")
}
