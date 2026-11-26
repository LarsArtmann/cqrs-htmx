package setup

import (
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// validateAuditStores rejects audit stores the dashboard cannot render. The
// audit panels list the complete trail, so a store that cannot read it all
// (a sink-only store, or one without a ReadAll) would mount a panel that
// stays empty or fails on its first load. With the dashboard disabled the
// stores would be a silent no-op. Both are the silent-misconfiguration class
// this validator family exists for.
func (c Config) validateAuditStores() error {
	if c.DisableDashboard && (c.CommandStore != nil || c.QueryStore != nil) {
		return errorfamily.NewRejection(
			"setup.invalid_config",
			"CommandStore and QueryStore feed the dashboard's audit panels, so they are ignored while DisableDashboard is set — drop them or enable the dashboard",
		)
	}

	if c.CommandStore != nil {
		if _, ok := c.CommandStore.(command.CommandJournal); !ok {
			return errorfamily.NewRejection(
				"setup.invalid_config",
				"CommandStore cannot list the whole command trail (it does not implement command.CommandJournal), so the dashboard's command audit panel cannot render it — pass the store go-cqrs-lite's system builds (system.CommandStore()) or any store with a ReadAll method",
			)
		}
	}

	if c.QueryStore != nil {
		if _, ok := c.QueryStore.(query.QueryJournal); !ok {
			return errorfamily.NewRejection(
				"setup.invalid_config",
				"QueryStore cannot list the whole query trail (it does not implement query.QueryJournal), so the dashboard's query audit panel cannot render it — pass the store go-cqrs-lite's system builds (system.QueryStore()) or any store with a ReadAllQueries method",
			)
		}
	}

	return nil
}
