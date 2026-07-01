package cqrshtmx_test

import (
	"context"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
)

type testListUsersQuery struct {
	*query.BasicQuery
}

func newTestListUsersQuery() *testListUsersQuery {
	core, err := query.New("ListUsers")
	if err != nil {
		panic(err)
	}
	return &testListUsersQuery{BasicQuery: core}
}

// testGetUserNameQuery is used by the ExampleApp_Query_typedDispatch example.
type testGetUserNameQuery struct {
	*query.BasicQuery
}

func newTestGetUserNameQuery() *testGetUserNameQuery {
	core, err := query.New("GetUserName")
	if err != nil {
		panic(err)
	}
	return &testGetUserNameQuery{BasicQuery: core}
}

// --- Decoder helpers ---

func decodeCreateUserJSON() cqrshtmx.HandlerOption {
	return cqrshtmx.DecodeJSON(func(_ testCreateUserRequest) (command.Command, error) {
		return &testCreateUserCmd{aggID: id.NewAggregateID(), cmdID: id.NewCommandID()}, nil
	})
}

func decodeCreateUserJSONWithBody() cqrshtmx.HandlerOption {
	return cqrshtmx.DecodeJSON(testCreateUserCommand)
}

func decodeBDDCreateUserJSON() cqrshtmx.HandlerOption {
	return cqrshtmx.DecodeJSON(func(_ bddCreateUserReq) (command.Command, error) {
		return &bddCreateUserCmd{aggID: id.NewAggregateID(), cmdID: id.NewCommandID()}, nil
	})
}

func decodeGetUserJSONQuery() cqrshtmx.HandlerOption {
	return cqrshtmx.DecodeJSONQuery(func(_ testGetUserQuery) (query.Query, error) {
		return &testGetUserQuery{}, nil
	})
}

func decodeCreateUserJSONWithAggID(aggID id.AggregateID) cqrshtmx.HandlerOption {
	return cqrshtmx.DecodeJSON(func(_ testCreateUserRequest) (command.Command, error) {
		return &testCreateUserCmd{aggID: aggID, cmdID: id.NewCommandID()}, nil
	})
}

func decodeCreateUserJSONWithBodyAndAggID(aggID id.AggregateID) cqrshtmx.HandlerOption {
	return cqrshtmx.DecodeJSON(func(req testCreateUserRequest) (command.Command, error) {
		return &testCreateUserCmd{aggID: aggID, cmdID: id.NewCommandID(), email: req.Email, name: req.Name}, nil
	})
}

// bddCreateUserCommand builds a bddCreateUserCmd from a request body.
// Shared by JSON and form decoders; the aggID is fresh on every call.
func bddCreateUserCommand(req bddCreateUserReq) (command.Command, error) {
	return &bddCreateUserCmd{
		aggID: id.NewAggregateID(),
		cmdID: id.NewCommandID(),
		email: req.Email,
		name:  req.Name,
	}, nil
}

// testCreateUserCommand is the same pattern as bddCreateUserCommand but
// for the testCreateUserRequest / testCreateUserCmd pair.
func testCreateUserCommand(req testCreateUserRequest) (command.Command, error) {
	return &testCreateUserCmd{
		aggID: id.NewAggregateID(),
		cmdID: id.NewCommandID(),
		email: req.Email,
		name:  req.Name,
	}, nil
}

// registerListUsersQuery registers a "ListUsers" query handler that
// returns a fixed list of users. Used by the typed-dispatch example
// and benchmark to keep their boilerplate identical.
func registerListUsersQuery(disp *query.Dispatcher, users []string) error {
	return query.RegisterTyped(
		disp, "ListUsers",
		func(_ context.Context, _ *testListUsersQuery) ([]string, error) {
			return users, nil
		},
	)
}

func decodeBDDCreateUserJSONWithBody() cqrshtmx.HandlerOption {
	return cqrshtmx.DecodeJSON(bddCreateUserCommand)
}

func decodeBDDCreateUserFormMapper() func(bddCreateUserReq) (command.Command, error) {
	return bddCreateUserCommand
}

// --- Handler helpers ---
