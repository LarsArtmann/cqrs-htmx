package usermgmt

// BotRegisteredPayload is emitted when a new bot (machine actor) is registered
// with an API token. TokenHash is the HMAC-SHA256 of the raw token.
type BotRegisteredPayload struct {
	SchemaVersion int      `json:"schema_version"`
	Name          string   `json:"name"`
	OwnerID       UserID   `json:"owner_id"`
	TokenHash     []byte   `json:"token_hash"`
	Scopes        []string `json:"scopes"`
}

// BotDeletedPayload is emitted when a bot is permanently deleted.
type BotDeletedPayload struct {
	SchemaVersion int    `json:"schema_version"`
	Reason        string `json:"reason"`
}
