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
//
// The email is REQUIRED on RegisterUserCmd and ChangeEmailCmd: an empty or
// blank address fails validation exactly like a malformed one, so this seam
// agrees with the Service (ParseEmail) and with the deciders' blank-email
// guard. ChangeDisplayNameCmd carries no email; its display name is optional
// and only length-checked.
func ValidateCommand(cmd command.Command) error {
	switch c := cmd.(type) {
	case *RegisterUserCmd:
		return validateEmailAndDisplayName(c.Email(), c.DisplayName())
	case *ChangeEmailCmd:
		return validateEmailAndDisplayName(c.Email(), "")
	case *ChangeDisplayNameCmd:
		return validateOptionalDisplayName(c.DisplayName())
	default:
		return nil
	}
}

// validateEmailAndDisplayName applies the request-layer syntactic rules to a
// command whose email is required (RegisterUserCmd, ChangeEmailCmd). The email
// is always parsed — ParseEmail rejects an empty or blank address like a
// malformed one — so an empty email can no longer slip past this middleware
// only to be refused (or worse, stored) further down. The display name is
// optional on these commands: empty skips its check, only the length rule
// applies.
func validateEmailAndDisplayName(email, displayName string) error {
	var errs []string
	if _, err := ParseEmail(email); err != nil {
		errs = append(errs, "invalid email")
	}
	errs = appendDisplayNameLengthError(errs, displayName)
	return formatValidationErrors(errs)
}

// validateOptionalDisplayName applies the display-name length rule to a
// command whose only syntactic payload is the optional display name
// (ChangeDisplayNameCmd). An empty display name clears the name and is valid.
func validateOptionalDisplayName(displayName string) error {
	return formatValidationErrors(appendDisplayNameLengthError(nil, displayName))
}

// appendDisplayNameLengthError appends the length-rule message to errs when
// displayName exceeds maxDisplayNameLength.
func appendDisplayNameLengthError(errs []string, displayName string) []string {
	if len(displayName) > maxDisplayNameLength {
		errs = append(errs, "display name must be under "+strconv.Itoa(maxDisplayNameLength)+" characters")
	}
	return errs
}
