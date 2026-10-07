package systemadapter_test

import (
	"fmt"

	systemadapter "github.com/larsartmann/cqrs-htmx/systemadapter/v4"
)

// ExampleDomainConfig shows the one-call wiring: DomainConfig supplies the
// whole identity domain (4 deciders, 20 commands, 21 event types, declarative
// projections); a Recommended* preset supplies the infrastructure shape.
func ExampleDomainConfig() {
	domain := systemadapter.DomainConfig(
		systemadapter.WithCheckpointStore(nil), // nil = system.New's engine-backed default
	)
	deployment := systemadapter.RecommendedSQLiteDeployment("file:app.db")

	fmt.Println("decoder wired:", domain.ProjectionTypeDecoder != nil)
	fmt.Println("driver:", deployment.Engines["primary"].Driver)
	// Output:
	// decoder wired: true
	// driver: sqlite
}
