package dashboardui

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// BDD suite bootstrap (one per package). The module's older tests stay
// plain go test; new behavior specs land here as Ginkgo Describe/It blocks.
func TestDashboardUISuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DashboardUI Suite")
}
