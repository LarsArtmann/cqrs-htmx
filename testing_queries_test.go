package cqrshtmx_test

import (
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
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
		return &testCreateUserCmd{aggID: id.NewAggregateID()}, nil
	})
}

func decodeCreateUserJSONWithBody() cqrshtmx.HandlerOption {
	return cqrshtmx.DecodeJSON(func(req testCreateUserRequest) (command.Command, error) {
		return &testCreateUserCmd{aggID: id.NewAggregateID(), email: req.Email, name: req.Name}, nil
	})
}

func decodeBDDCreateUserJSON() cqrshtmx.HandlerOption {
	return cqrshtmx.DecodeJSON(func(_ bddCreateUserReq) (command.Command, error) {
		return &bddCreateUserCmd{aggID: id.NewAggregateID()}, nil
	})
}

func decodeGetUserJSONQuery() cqrshtmx.HandlerOption {
	return cqrshtmx.DecodeJSONQuery(func(_ testGetUserQuery) (query.Query, error) {
		return &testGetUserQuery{}, nil
	})
}

func decodeCreateUserJSONWithAggID(aggID id.AggregateID) cqrshtmx.HandlerOption {
	return cqrshtmx.DecodeJSON(func(_ testCreateUserRequest) (command.Command, error) {
		return &testCreateUserCmd{aggID: aggID}, nil
	})
}

func decodeCreateUserJSONWithBodyAndAggID(aggID id.AggregateID) cqrshtmx.HandlerOption {
	return cqrshtmx.DecodeJSON(func(req testCreateUserRequest) (command.Command, error) {
		return &testCreateUserCmd{aggID: aggID, email: req.Email, name: req.Name}, nil
	})
}

func decodeBDDCreateUserJSONWithBody() cqrshtmx.HandlerOption {
	return cqrshtmx.DecodeJSON(func(req bddCreateUserReq) (command.Command, error) {
		return &bddCreateUserCmd{
			aggID: id.NewAggregateID(),
			email: req.Email,
			name:  req.Name,
		}, nil
	})
}

func decodeBDDCreateUserFormMapper() func(bddCreateUserReq) (command.Command, error) {
	return func(req bddCreateUserReq) (command.Command, error) {
		return &bddCreateUserCmd{
			aggID: id.NewAggregateID(),
			email: req.Email,
			name:  req.Name,
		}, nil
	}
}

// --- Handler helpers ---
