package usermgmt

import brandid "github.com/larsartmann/go-branded-id"

type userBrand struct{}

type UserID = brandid.ID[userBrand, string]

func NewUserID(s string) UserID { return brandid.NewID[userBrand](s) }
