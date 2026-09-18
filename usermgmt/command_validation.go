package usermgmt

import (
	"strconv"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
)

// ValidateCommand checks the syntactic shape of identity-domain commands
// (email parseability, display-name length) before dispatch. It is the
// command-level twin of [RegisterRequest.Validate]: the Service methods
// already apply these rules to their requests, but direct dispatcher use
// bypasses them — wiring this via middleware.CommandValidation closes that
// gap and gives every dispatch path the same Rejection-family errors (HTTP
// 400) with the same messages and sentinel:
//
//	ServiceConfig{
//		CommandMiddleware: []command.Middleware{
//			middleware.CommandValidation(ValidateCommand),
//		},
//	}
//
// Commands without syntactic payload rules (credentials, TOTP secrets,
// opaque identifiers) pass through — their invariants are semantic and stay
// in the domain decide functions, which remain the authority for every
// command regardless of this middleware.
func ValidateCommand(cmd command.Command) error {
	switch c := cmd.(type) {
	case *RegisterUserCmd:
		return validateEmailAndDisplayName(c.Email(), c.DisplayName())
	case *ChangeEmailCmd:
		return validateEmailAndDisplayName(c.Email(), "")
	case *ChangeDisplayNameCmd:
		return validateEmailAndDisplayName("", c.DisplayName())
	default:
		return nil
	}
}

// validateEmailAndDisplayName applies the request-layer syntactic rules.
// Empty arguments skip their check: the middleware validates only the
// payload fields a command actually carries.
func validateEmailAndDisplayName(email, displayName string) error {
	var errs []string
	if email != "" {
		if _, err := ParseEmail(email); err != nil {
			errs = append(errs, "invalid email")
		}
	}
	if len(displayName) > maxDisplayNameLength {
		errs = append(errs, "display name must be under "+strconv.Itoa(maxDisplayNameLength)+" characters")
	}
	return formatValidationErrors(errs)
}
