package usermgmt

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	bcryptCost = minBcryptCost
	os.Exit(m.Run())
}
