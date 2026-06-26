package usermgmt

import (
	"encoding/json"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// marshalViewJSON marshals an aggregate entity to JSON for inclusion in a SQL
// view's Data blob. On failure it wraps the encoding error with the
// infrastructure family so callers can return it directly. errCode is the
// stable metric/log code (e.g. "usermgmt.sql_readmodel.tenant_marshal"); msg
// is the human-readable label (e.g. "marshal tenant data").
//
// Centralises the encode-or-wrap idiom shared by every SQL-backed read model
// (User, Membership, Tenant, Bot).
func marshalViewJSON[T any](entity T, errCode, msg string) (string, error) {
	data, err := json.Marshal(entity)
	if err != nil {
		return "", event.WrapInfrastructure(err, errCode, msg)
	}
	return string(data), nil
}
