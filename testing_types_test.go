package cqrshtmx_test

import (
	"context"
	"io"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

type testCreateUserCmd struct {
	aggID id.AggregateID
	cmdID id.CommandID
	email string
	name  string
}

func (c *testCreateUserCmd) Type() command.Type          { return "CreateUser" }
func (c *testCreateUserCmd) AggregateID() id.AggregateID { return c.aggID }
func (c *testCreateUserCmd) ID() id.CommandID            { return c.cmdID }
func (c *testCreateUserCmd) IdempotencyKey() string      { return c.aggID.String() }

type testCreateUserRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type testGetUserQuery struct{}

func (q *testGetUserQuery) Type() query.Type { return "GetUser" }

type bddCreateUserReq struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type bddCreateUserCmd struct {
	aggID id.AggregateID
	cmdID id.CommandID
	email string
	name  string
}

func (c *bddCreateUserCmd) Type() command.Type          { return "CreateUser" }
func (c *bddCreateUserCmd) AggregateID() id.AggregateID { return c.aggID }
func (c *bddCreateUserCmd) ID() id.CommandID            { return c.cmdID }
func (c *bddCreateUserCmd) IdempotencyKey() string      { return c.aggID.String() }

type bddDeleteUserCmd struct {
	aggID id.AggregateID
	cmdID id.CommandID
}

func (c *bddDeleteUserCmd) Type() command.Type          { return "DeleteUser" }
func (c *bddDeleteUserCmd) AggregateID() id.AggregateID { return c.aggID }
func (c *bddDeleteUserCmd) ID() id.CommandID            { return c.cmdID }
func (c *bddDeleteUserCmd) IdempotencyKey() string      { return c.aggID.String() }

type bddListUsersQuery struct{}

func (q *bddListUsersQuery) Type() query.Type { return "ListUsers" }

type bddDashboardQuery struct{}

func (q *bddDashboardQuery) Type() query.Type { return "GetDashboard" }

type bddTemplComponent struct {
	html string
}

func (m *bddTemplComponent) Render(_ context.Context, w io.Writer) error {
	_, err := w.Write([]byte(m.html))

	return err
}

type getPageQuery struct{}

func (q *getPageQuery) Type() query.Type { return "GetPage" }

// testListUsersQuery is used by the ExampleApp_Query_typedRegister example.
// It embeds *query.BasicQuery to inherit Type() and be dispatchable via
