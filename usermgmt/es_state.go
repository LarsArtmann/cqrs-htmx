package usermgmt

import (
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

type UserState = identitymodel.UserState

func unmarshalPayload[T any](evt event.Event) (T, error) {
	return identitymodel.UnmarshalPayload[T](evt)
}

var foldUser = identitymodel.FoldUser
