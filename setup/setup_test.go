package setup_test

// The setup package tests were split by concern (defaults, validation, paths,
// lifecycle) into four focused files:
//
//   - setup_defaults_test.go: zero-config behaviour, feature-flag defaults,
//     non-path/non-validation Config field passthroughs.
//   - setup_validation_test.go: Config fields that setup.New/MustNew must
//     reject (path syntax, conflicts, root reservation, cross-field rules).
//   - setup_paths_test.go: path resolution, mounting, BasePath wiring, auth
//     gates, and trailing-slash normalisation.
//   - setup_lifecycle_test.go: MustNew, middleware exposure, Close idempotency,
//     Run, AsyncStartup, SyncStartup.
//
// This file is intentionally empty — the package exists so each test file can
// share the same `setup_test` external test package without redeclaration.
