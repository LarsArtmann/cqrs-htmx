package usermgmt

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
)

func TestImportExport_JSONRoundTrip(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	ctx := context.Background()

	importData := []ImportUser{
		{Email: "imp1@test.com", DisplayName: "Imp One"},
		{Email: "imp2@test.com", DisplayName: "Imp Two"},
		{Email: "imp3@test.com"},
	}
	body, err := json.Marshal(importData)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	result, err := svc.ImportUsersFromJSON(ctx, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("ImportUsersFromJSON: %v", err)
	}
	if result.Imported != 3 {
		t.Errorf("imported = %d, want 3", result.Imported)
	}

	// Export and verify
	var buf bytes.Buffer
	if err := svc.ExportUsersToJSON(ctx, &buf); err != nil {
		t.Fatalf("ExportUsersToJSON: %v", err)
	}

	var exported []ExportUser
	if err := json.Unmarshal(buf.Bytes(), &exported); err != nil {
		t.Fatalf("unmarshal export: %v", err)
	}
	if len(exported) != 3 {
		t.Fatalf("exported %d users, want 3", len(exported))
	}
}

func TestImportExport_JSONSkipsExistingEmails(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	ctx := context.Background()
	registerTestUser(t, svc, "dup1", "dup1@test.com")

	importData := []ImportUser{
		{Email: "dup1@test.com"}, // already exists
		{Email: "new1@test.com"},
	}
	body, err := json.Marshal(importData)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	result, err := svc.ImportUsersFromJSON(ctx, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("ImportUsersFromJSON: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("imported = %d, want 1", result.Imported)
	}
	if result.Skipped != 1 {
		t.Errorf("skipped = %d, want 1", result.Skipped)
	}
}

func TestImportExport_CSVImport(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	ctx := context.Background()

	csvData := "email,display_name\ncsv1@test.com,CSV One\ncsv2@test.com,CSV Two\n"
	result, err := svc.ImportUsersFromCSV(ctx, strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("ImportUsersFromCSV: %v", err)
	}
	if result.Imported != 2 {
		t.Errorf("imported = %d, want 2", result.Imported)
	}

	// Verify user was created
	if _, ok := svc.readModel.FindByEmail("csv1@test.com"); !ok {
		t.Error("csv1@test.com not found")
	}
}

func TestImportExport_CSVExport(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	ctx := context.Background()
	registerTestUser(t, svc, "exp1", "exp1@test.com")

	var buf bytes.Buffer
	if err := svc.ExportUsersToCSV(ctx, &buf); err != nil {
		t.Fatalf("ExportUsersToCSV: %v", err)
	}

	reader := csv.NewReader(&buf)
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("read CSV: %v", err)
	}
	if len(rows) < 2 {
		t.Fatalf("expected header + at least 1 row, got %d rows", len(rows))
	}
	if rows[0][0] != "id" || rows[0][1] != "email" {
		t.Errorf("unexpected header: %v", rows[0])
	}
}

func TestImportExport_EmptyEmailSkipped(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	ctx := context.Background()

	importData := []ImportUser{
		{Email: ""},
		{Email: "valid@test.com"},
	}
	body, err := json.Marshal(importData)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	result, err := svc.ImportUsersFromJSON(ctx, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("ImportUsersFromJSON: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("imported = %d, want 1", result.Imported)
	}
	if result.Skipped != 1 {
		t.Errorf("skipped = %d, want 1", result.Skipped)
	}
}

func TestImportExport_InvalidJSON(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	ctx := context.Background()

	_, err := svc.ImportUsersFromJSON(ctx, strings.NewReader("{invalid"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestImportExport_CSVWithFlexibleHeaders(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	ctx := context.Background()

	// Test with "name" instead of "display_name"
	csvData := "email,name\nflex@test.com,Flexible User\n"
	result, err := svc.ImportUsersFromCSV(ctx, strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("ImportUsersFromCSV: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("imported = %d, want 1", result.Imported)
	}

	user, ok := svc.readModel.FindByEmail("flex@test.com")
	if !ok {
		t.Fatal("user not found")
	}
	if user.DisplayName != "Flexible User" {
		t.Errorf("display_name = %q, want %q", user.DisplayName, "Flexible User")
	}
}
