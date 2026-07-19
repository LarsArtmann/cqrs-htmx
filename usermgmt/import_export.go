package usermgmt

import (
	"context"
	"encoding/csv"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// ImportUser represents a single user to import.
type ImportUser struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name,omitempty"`
}

// Validate normalizes and validates the import user fields.
// Returns ErrValidation if any field is invalid.
func (u *ImportUser) Validate() error {
	u.DisplayName = strings.TrimSpace(u.DisplayName)
	email, err := ParseEmail(u.Email)
	if err != nil {
		return err
	}
	u.Email = email.String()
	if len(u.DisplayName) > maxDisplayNameLength {
		return errorfamily.Wrapf(
			ErrValidation,
			event.Rejection,
			"usermgmt.import.display_name_too_long",
			"display name too long (max %d)",
			maxDisplayNameLength,
		)
	}
	return nil
}

// ExportUser is a user in the export format. It omits credentials and
// sensitive data — only public profile information is included.
// Roles are not included: query them via Service.Authz().RolesForUser().
type ExportUser struct {
	ID               UserID            `json:"id"`
	Email            Email             `json:"email"`
	DisplayName      string            `json:"display_name,omitempty"`
	EmailVerified    bool              `json:"email_verified"`
	TOTPEnabled      bool              `json:"totp_enabled"`
	ExternalAccounts []ExternalAccount `json:"external_accounts,omitempty"`
}

// ImportResult reports the outcome of a batch import.
type ImportResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors,omitempty"`
}

// UserDataFormat is the serialization format for user data import/export.
type UserDataFormat string

const (
	UserDataFormatJSON UserDataFormat = "json"
	UserDataFormatCSV  UserDataFormat = "csv"
)

// Valid reports whether f is one of the declared UserDataFormat constants.
// Use this at trust boundaries (e.g. HTTP Accept/Header decode) to reject
// unknown formats before dispatching the importer.
func (f UserDataFormat) Valid() bool {
	switch f {
	case UserDataFormatJSON, UserDataFormatCSV:
		return true
	}
	return false
}

// ImportUsersFromJSON reads a JSON array of ImportUser from the reader and
// registers each user. Users with existing emails are skipped.
// Returns a summary of imported/skipped counts and any per-user errors.
func (s *Service) ImportUsersFromJSON(ctx context.Context, r io.Reader) (*ImportResult, error) {
	var users []ImportUser
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, errorfamily.WrapRejection(err, "usermgmt.import.json_read_failed", "read import JSON")
	}
	if err := json.Unmarshal(body, &users); err != nil {
		return nil, errorfamily.WrapRejection(err, "usermgmt.import.json_decode_failed", "decode import JSON")
	}
	return s.importUsers(ctx, users)
}

// ImportUsersFromCSV reads CSV data with columns: email, display_name.
// The header row is required. Users with existing emails are skipped.
func (s *Service) ImportUsersFromCSV(ctx context.Context, r io.Reader) (*ImportResult, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1 // Allow variable field count

	rows, err := reader.ReadAll()
	if err != nil {
		return nil, errorfamily.WrapRejection(err, "usermgmt.import.csv_read_failed", "read CSV")
	}
	if len(rows) == 0 {
		return &ImportResult{ //nolint:exhaustruct // zero-value is intentional for empty import
			Imported: 0,
			Skipped:  0,
		}, nil
	}

	header := rows[0]
	emailIdx, nameIdx := findCSVColumns(header)

	users := make([]ImportUser, 0, len(rows)-1)
	for _, row := range rows[1:] {
		if len(row) == 0 {
			continue
		}
		u := ImportUser{
			Email:       "",
			DisplayName: "",
		}
		if emailIdx >= 0 && emailIdx < len(row) {
			u.Email = strings.TrimSpace(row[emailIdx])
		}
		if nameIdx >= 0 && nameIdx < len(row) {
			u.DisplayName = strings.TrimSpace(row[nameIdx])
		}
		if u.Email == "" {
			continue
		}
		users = append(users, u)
	}

	return s.importUsers(ctx, users)
}

func (s *Service) importUsers(ctx context.Context, users []ImportUser) (*ImportResult, error) {
	result := &ImportResult{Errors: []string{}, Imported: 0, Skipped: 0}
	for i := range users {
		if err := users[i].Validate(); err != nil {
			result.Skipped++
			result.Errors = append(result.Errors, err.Error())
			continue
		}
		if _, ok := s.readModel.FindByEmail(users[i].Email); ok {
			result.Skipped++
			continue
		}

		aggID := id.NewAggregateID()
		if err := s.dispatcher.Dispatch(ctx, NewRegisterUserCmd(
			aggID, users[i].Email, users[i].DisplayName, []Role{RoleUser},
		)); err != nil {
			result.Errors = append(result.Errors,
				fmt.Sprintf("%s: %v", users[i].Email, err))
			continue
		}
		result.Imported++
	}
	if len(result.Errors) == 0 {
		result.Errors = nil
	}
	return result, nil
}

// ExportUsersToJSON writes all users as a JSON array to the writer.
func (s *Service) ExportUsersToJSON(_ context.Context, w io.Writer) error {
	users := s.exportAllUsers()
	enc := jsontext.NewEncoder(w, jsontext.WithIndent("  "))
	if err := json.MarshalEncode(enc, users); err != nil {
		return errorfamily.WrapInfrastructure(err, "usermgmt.export.json_encode_failed", "encode export JSON")
	}
	return nil
}

const (
	csvColumnID            = "id"
	csvColumnEmail         = "email"
	csvColumnEmailAlt      = "e-mail"
	csvColumnDisplayName   = "display_name"
	csvColumnName          = "name"
	csvColumnNameAlt       = "displayname"
	csvColumnEmailVerified = "email_verified"
	csvColumnTOTPEnabled   = "totp_enabled"
)

// ExportUsersToCSV writes all users as CSV with columns: id, email, display_name, email_verified, totp_enabled.
func (s *Service) ExportUsersToCSV(_ context.Context, w io.Writer) error {
	users := s.exportAllUsers()
	cw := csv.NewWriter(w)
	defer cw.Flush()

	if err := cw.Write([]string{
		csvColumnID,
		csvColumnEmail,
		csvColumnDisplayName,
		csvColumnEmailVerified,
		csvColumnTOTPEnabled,
	}); err != nil {
		return errorfamily.WrapTransient(err, "usermgmt.export.csv_header_failed", "write CSV header")
	}
	for _, u := range users {
		if err := cw.Write([]string{
			u.ID.Get().String(),
			string(u.Email),
			u.DisplayName,
			strconv.FormatBool(u.EmailVerified),
			strconv.FormatBool(u.TOTPEnabled),
		}); err != nil {
			return errorfamily.WrapTransient(err, "usermgmt.export.csv_row_failed", "write CSV row")
		}
	}
	return nil
}

func (s *Service) exportAllUsers() []ExportUser {
	all := s.readModel.AllUsers()
	users := make([]ExportUser, 0, len(all))
	for _, u := range all {
		users = append(users, ExportUser{
			ID:               u.ID,
			Email:            Email(u.Email),
			DisplayName:      u.DisplayName,
			EmailVerified:    u.EmailVerified,
			TOTPEnabled:      u.TOTPEnabled,
			ExternalAccounts: append([]ExternalAccount(nil), u.ExternalAccounts...),
		})
	}
	return users
}

func findCSVColumns(header []string) (emailIdx, nameIdx int) {
	emailIdx, nameIdx = -1, -1
	for i, col := range header {
		switch strings.ToLower(strings.TrimSpace(col)) {
		case csvColumnEmail, csvColumnEmailAlt:
			emailIdx = i
		case csvColumnDisplayName, csvColumnName, csvColumnNameAlt:
			nameIdx = i
		}
	}
	return emailIdx, nameIdx
}
