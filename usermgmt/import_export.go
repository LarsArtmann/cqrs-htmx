package usermgmt

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/mail"
	"strconv"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// ImportUser represents a single user to import.
type ImportUser struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name,omitempty"`
}

// Validate normalizes and validates the import user fields.
// Returns ErrValidation if any field is invalid.
func (u *ImportUser) Validate() error {
	u.Email = strings.ToLower(strings.TrimSpace(u.Email))
	u.DisplayName = strings.TrimSpace(u.DisplayName)
	if u.Email == "" {
		return fmt.Errorf("%w: email is required", ErrValidation)
	}
	if _, err := mail.ParseAddress(u.Email); err != nil {
		return fmt.Errorf("%w: invalid email %q: %s", ErrValidation, u.Email, err)
	}
	if len(u.Email) > maxEmailLength {
		return fmt.Errorf("%w: email too long (max %d)", ErrValidation, maxEmailLength)
	}
	if len(u.DisplayName) > maxDisplayNameLength {
		return fmt.Errorf("%w: display name too long (max %d)", ErrValidation, maxDisplayNameLength)
	}
	return nil
}

// ExportUser is a user in the export format. It omits credentials and
// sensitive data — only public profile information is included.
type ExportUser struct {
	ID            UserID `json:"id"`
	Email         string `json:"email"`
	DisplayName   string `json:"display_name,omitempty"`
	Roles         []Role `json:"roles"`
	EmailVerified bool   `json:"email_verified"`
	TOTPEnabled   bool   `json:"totp_enabled"`
}

// ImportResult reports the outcome of a batch import.
type ImportResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors,omitempty"`
}

// ExportFormat is the output format for user data export.
type ExportFormat string

const (
	ExportFormatJSON ExportFormat = "json"
	ExportFormatCSV  ExportFormat = "csv"
)

// ImportUsersFromJSON reads a JSON array of ImportUser from the reader and
// registers each user. Users with existing emails are skipped.
// Returns a summary of imported/skipped counts and any per-user errors.
func (s *Service) ImportUsersFromJSON(ctx context.Context, r io.Reader) (*ImportResult, error) {
	var users []ImportUser
	if err := json.NewDecoder(r).Decode(&users); err != nil {
		return nil, fmt.Errorf("decode import JSON: %w", err)
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
		return nil, fmt.Errorf("read CSV: %w", err)
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
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(users); err != nil {
		return fmt.Errorf("encode export JSON: %w", err)
	}
	return nil
}

const (
	csvColumnID            = "id"
	csvColumnEmail         = "email"
	csvColumnDisplayName   = "display_name"
	csvColumnRoles         = "roles"
	csvColumnEmailVerified = "email_verified"
	csvColumnTOTPEnabled   = "totp_enabled"
)

// ExportUsersToCSV writes all users as CSV with columns: id, email, display_name, roles, email_verified, totp_enabled.
func (s *Service) ExportUsersToCSV(_ context.Context, w io.Writer) error {
	users := s.exportAllUsers()
	cw := csv.NewWriter(w)
	defer cw.Flush()

	if err := cw.Write([]string{
		csvColumnID,
		csvColumnEmail,
		csvColumnDisplayName,
		csvColumnRoles,
		csvColumnEmailVerified,
		csvColumnTOTPEnabled,
	}); err != nil {
		return fmt.Errorf("write CSV header: %w", err)
	}
	for _, u := range users {
		roles := make([]string, 0, len(u.Roles))
		for _, r := range u.Roles {
			roles = append(roles, string(r))
		}
		if err := cw.Write([]string{
			u.ID.Get(),
			u.Email,
			u.DisplayName,
			strings.Join(roles, ";"),
			strconv.FormatBool(u.EmailVerified),
			strconv.FormatBool(u.TOTPEnabled),
		}); err != nil {
			return fmt.Errorf("write CSV row: %w", err)
		}
	}
	return nil
}

func (s *Service) exportAllUsers() []ExportUser {
	all := s.readModel.AllUsers()
	users := make([]ExportUser, 0, len(all))
	for _, u := range all {
		users = append(users, ExportUser{
			ID:            u.ID,
			Email:         u.Email,
			DisplayName:   u.DisplayName,
			Roles:         append([]Role(nil), u.Roles...),
			EmailVerified: u.EmailVerified,
			TOTPEnabled:   u.TOTPEnabled,
		})
	}
	return users
}

func findCSVColumns(header []string) (emailIdx, nameIdx int) {
	emailIdx, nameIdx = -1, -1
	for i, col := range header {
		switch strings.ToLower(strings.TrimSpace(col)) {
		case "email", "e-mail":
			emailIdx = i
		case "display_name", "name", "displayname":
			nameIdx = i
		}
	}
	return emailIdx, nameIdx
}
