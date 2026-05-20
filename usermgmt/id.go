package usermgmt

import brandid "github.com/larsartmann/go-branded-id"

type userBrand struct{}

// UserID is a branded type for user identifiers backed by ULID strings.
// Use NewUserID to construct instances from string values.
type UserID = brandid.ID[userBrand, string]

// NewUserID constructs a UserID from a string value.
// In tests, prefer passing a known ULID string for determinism.
func NewUserID(s string) UserID { return brandid.NewID[userBrand](s) }
