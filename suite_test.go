package cqrshtmx_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCQRSHTMX(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CQRS-HTMX Suite")
}
