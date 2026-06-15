package usermgmt

import "encoding/json"

type UserRegisteredPayload struct {
	Email        string `json:"email"`
	DisplayName  string `json:"display_name,omitempty"`
	PasswordHash string `json:"password_hash"`
	Roles        []Role `json:"roles"`
}

type PasswordChangedPayload struct {
	PasswordHash string `json:"password_hash"`
}

type RolesUpdatedPayload struct {
	Roles  []Role `json:"roles"`
	Domain string `json:"domain"`
}

type EmailChangedPayload struct {
	Email string `json:"email"`
}

type DisplayNameChangedPayload struct {
	DisplayName string `json:"display_name"`
}

type UserDeletedPayload struct {
	Reason string `json:"reason"`
}

func marshalPayload(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
