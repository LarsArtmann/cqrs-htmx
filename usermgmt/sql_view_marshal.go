package usermgmt

import (
	"encoding/json/v2"

	errorfamily "github.com/larsartmann/go-error-family"
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
		return "", errorfamily.WrapInfrastructure(err, errCode, msg)
	}
	return string(data), nil
}

// unmarshalViewJSON unmarshals a SQL view's Data blob back into an entity,
// wrapping decode failures as Corruption errors (the blob is derived state —
// an undecodable row means the view store drifted from the code). errCode is
// the stable metric/log code; msg is the human-readable label. The inverse of
// marshalViewJSON, used by the read-model Hydrate methods.
func unmarshalViewJSON[T any](data, errCode, msg string) (T, error) {
	var entity T
	if err := json.Unmarshal([]byte(data), &entity); err != nil {
		return entity, errorfamily.WrapCorruption(err, errCode, msg)
	}
	return entity, nil
}

// wrapTransientOrOK returns nil when err is nil, otherwise wraps err as a
// Transient failure with the caller's error code and message. Eliminates the
// repeated `if err != nil { return WrapTransient(err, code, msg) }; return nil`
// boilerplate shared across SQL store methods.
func wrapTransientOrOK(err error, errCode, errMsg string) error {
	if err == nil {
		return nil
	}
	return errorfamily.WrapTransient(err, errCode, errMsg)
}
