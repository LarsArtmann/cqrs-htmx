package setup

import errorfamily "github.com/larsartmann/go-error-family"

// validateTracking rejects the tracking configurations New would not honor.
// Two homes for the same signals have no defined winner, and the shell
// (DisableService) mounts no tracking endpoint at all, so a sink or a backend
// handed to it would store nothing while looking configured. Both fail at New
// with the way out in the message.
func (c Config) validateTracking() error {
	switch {
	case c.TrackingSink != nil && c.TrackingBackend != nil:
		return errorfamily.NewRejection(
			"setup.invalid_config",
			"TrackingSink and TrackingBackend name two homes for the same signals; keep one: TrackingSink stores signals in a store of yours, TrackingBackend keeps them in the bundle's signals journal",
		)
	case c.DisableService && (c.TrackingSink != nil || c.TrackingBackend != nil):
		return errorfamily.NewRejection(
			"setup.invalid_config",
			"DisableService mounts no tracking endpoint, so TrackingSink and TrackingBackend would store nothing; mount cqrshtmx.TrackHandler over setup.NewSignalsJournal(backend) yourself, or drop both for the shell",
		)
	}

	return nil
}
