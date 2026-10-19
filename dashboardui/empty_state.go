package dashboardui

// The empty-state copy surface (cqrs-htmx#19): every static empty-state
// panel in the dashboard carries upstream copy keyed by an EmptyStateKey.
// Hosts override per key via Config.EmptyStateCopy; field-wise merging
// keeps upstream titles when an override only rebrands the message.

// EmptyStateKey identifies one static empty-state panel. Dynamic
// empty states that interpolate record identifiers (a missing snapshot for
// a specific stream, per-projection dead-letter lists) are not brandable —
// their copy is generated per request.
type EmptyStateKey string

const (
	// EmptyStateEvents is the events list with no filter active.
	EmptyStateEvents EmptyStateKey = "events"
	// EmptyStateEventsFiltered is the events list with an active filter
	// that matched nothing.
	EmptyStateEventsFiltered EmptyStateKey = "events-filtered"
	// EmptyStateCommands is the command audit list.
	EmptyStateCommands EmptyStateKey = "commands"
	// EmptyStateQueries is the query audit list.
	EmptyStateQueries EmptyStateKey = "queries"
	// EmptyStateProjections is the projections list.
	EmptyStateProjections EmptyStateKey = "projections"
	// EmptyStateSnapshots is the snapshots list.
	EmptyStateSnapshots EmptyStateKey = "snapshots"
	// EmptyStateDeadLetters is the dead-letter queue index.
	EmptyStateDeadLetters EmptyStateKey = "dead-letters"
	// EmptyStateTimeTravelEvents is the time-travel version timeline of a
	// selected aggregate with no recorded events.
	EmptyStateTimeTravelEvents EmptyStateKey = "timetravel-events"
	// EmptyStateTimeTravelAggregates is the time-travel aggregate picker.
	EmptyStateTimeTravelAggregates EmptyStateKey = "timetravel-aggregates"
	// EmptyStateSnapshotAggregates is the snapshots aggregate picker.
	EmptyStateSnapshotAggregates EmptyStateKey = "snapshot-aggregates"
)

// EmptyStateCopy is the brand-overridable text of one empty-state panel.
// Empty fields keep the upstream default for that field.
type EmptyStateCopy struct {
	Title   string
	Message string
}

// defaultEmptyStates is the upstream vocabulary — the copy every panel
// rendered before the config surface existed.
var defaultEmptyStates = map[EmptyStateKey]EmptyStateCopy{ //nolint:gochecknoglobals // upstream defaults, single package-private map like auditActions
	EmptyStateEvents: {
		Title:   "No events yet",
		Message: "Events will appear here as they are committed to the store.",
	},
	EmptyStateEventsFiltered: {
		Title:   "No matching events",
		Message: "No events match the current filters. Try adjusting or clearing them.",
	},
	EmptyStateCommands: {
		Title:   "No commands recorded",
		Message: "Commands will appear here as they are dispatched.",
	},
	EmptyStateQueries: {
		Title:   "No queries recorded",
		Message: "Queries will appear here as they are executed.",
	},
	EmptyStateProjections: {
		Title:   "No projections registered",
		Message: "",
	},
	EmptyStateSnapshots: {
		Title:   "No snapshot",
		Message: "",
	},
	EmptyStateDeadLetters: {
		Title:   "Dead-Letter Queue",
		Message: "No projections registered. Dead letters will appear here when projection errors occur.",
	},
	EmptyStateTimeTravelEvents: {
		Title:   "No events",
		Message: "",
	},
	EmptyStateTimeTravelAggregates: {
		Title:   "No aggregates found",
		Message: "Configure a StreamReader to list aggregates for time-travel inspection.",
	},
	EmptyStateSnapshotAggregates: {
		Title:   "No aggregates found",
		Message: "Configure a StreamReader to browse snapshots by aggregate.",
	},
}

// validateEmptyStateCopy rejects unknown keys — a mistyped map key would
// otherwise silently do nothing.
func validateEmptyStateCopy(overrides map[EmptyStateKey]EmptyStateCopy) error {
	for k := range overrides {
		if _, ok := defaultEmptyStates[k]; !ok {
			return errConfig("Config.EmptyStateCopy has unknown key " + string(k))
		}
	}

	return nil
}

// resolveEmptyStates merges the configured overrides field-wise over the
// upstream defaults. The result is a fresh map; callers may hold it for
// the dashboard's lifetime.
func (config Config) resolveEmptyStates() map[EmptyStateKey]EmptyStateCopy {
	resolved := make(map[EmptyStateKey]EmptyStateCopy, len(defaultEmptyStates))

	for k, def := range defaultEmptyStates {
		resolved[k] = def
	}

	for k, override := range config.EmptyStateCopy {
		merged := resolved[k]

		if override.Title != "" {
			merged.Title = override.Title
		}

		if override.Message != "" {
			merged.Message = override.Message
		}

		resolved[k] = merged
	}

	return resolved
}
