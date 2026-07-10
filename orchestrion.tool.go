// This file was created by `orchestrion pin`, and is used to ensure the
// `go.mod` file contains the necessary entries to ensure repeatable builds when
// using `orchestrion`. It is also used to set up which integrations are enabled.

//go:build tools

//go:generate go run github.com/DataDog/orchestrion pin -generate

package tools

// Imports in this file determine which tracer integrations are enabled in
// orchestrion. New integrations can be automatically discovered by running
// `orchestrion pin` again. You can also manually add new imports here to
// enable additional integrations. When doing so, you can run `orchestrion pin`
// to make sure manually added integrations are valid (i.e, the imported package
// includes a valid `orchestrion.yml` file).
import (
	// Ensures `orchestrion` is present in `go.mod` so that builds are repeatable.
	// Do not remove.
	_ "github.com/DataDog/orchestrion" // integration

	_ "github.com/DataDog/dd-trace-go/contrib/aws/aws-sdk-go-v2/v2/aws" // integration
	_ "github.com/DataDog/dd-trace-go/contrib/aws/aws-sdk-go/v2/aws"    // integration

	_ "github.com/DataDog/dd-trace-go/contrib/database/sql/v2" // integration

	_ "github.com/DataDog/dd-trace-go/contrib/go-chi/chi.v5/v2" // integration
	_ "github.com/DataDog/dd-trace-go/contrib/go-chi/chi/v2"    // integration

	_ "github.com/DataDog/dd-trace-go/contrib/jackc/pgx.v5/v2" // integration

	_ "github.com/DataDog/dd-trace-go/contrib/log/slog/v2" // integration

	_ "github.com/DataDog/dd-trace-go/contrib/net/http/v2" // integration

	_ "github.com/DataDog/dd-trace-go/v2/contrib/os" // integration

	_ "github.com/DataDog/dd-trace-go/v2/ddtrace/tracer" // integration
	_ "github.com/DataDog/dd-trace-go/v2/orchestrion"    // integration
	_ "github.com/DataDog/dd-trace-go/v2/profiler"       // integration
)
