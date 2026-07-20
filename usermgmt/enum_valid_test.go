package usermgmt

import "testing"

func TestAction_Valid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		action Action
		want   bool
	}{
		{ActionExecute, true},
		{ActionRead, true},
		{ActionAll, true},
		{Action("execute"), true},
		{Action(""), false},
		{Action("write"), false},   // plausible typo
		{Action("EXECUTE"), false}, // case-sensitive
		{Action("admin"), false},   // role mistaken for action
	}
	for _, tt := range cases {
		if got := tt.action.Valid(); got != tt.want {
			t.Errorf("Action(%q).Valid() = %v, want %v", tt.action, got, tt.want)
		}
	}
}

func TestEffect_Valid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		effect Effect
		want   bool
	}{
		{EffectAllow, true},
		{EffectDeny, true},
		{Effect("allow"), true},
		{Effect("deny"), true},
		{Effect(""), false},
		{Effect("ALLOW"), false}, // case-sensitive
		{Effect("permit"), false},
		{Effect("block"), false},
	}
	for _, tt := range cases {
		if got := tt.effect.Valid(); got != tt.want {
			t.Errorf("Effect(%q).Valid() = %v, want %v", tt.effect, got, tt.want)
		}
	}
}

func TestRole_Valid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		role Role
		want bool
	}{
		{RoleSuperAdmin, true},
		{RoleAdmin, true},
		{RoleUser, true},
		{RoleViewer, true},
		{RoleOwner, true},
		{Role("admin"), true},
		{Role(""), false},
		{Role("super-admin"), false}, // hyphen vs underscore
		{Role("ADMIN"), false},       // case-sensitive
		{Role("manager"), false},     // plausible but not declared
		{Role("guest"), false},
	}
	for _, tt := range cases {
		if got := tt.role.Valid(); got != tt.want {
			t.Errorf("Role(%q).Valid() = %v, want %v", tt.role, got, tt.want)
		}
	}
}

func TestUserDataFormat_Valid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		format UserDataFormat
		want   bool
	}{
		{UserDataFormatJSON, true},
		{UserDataFormatCSV, true},
		{UserDataFormat("json"), true},
		{UserDataFormat("csv"), true},
		{UserDataFormat(""), false},
		{UserDataFormat("JSON"), false}, // case-sensitive
		{UserDataFormat("xml"), false},
		{UserDataFormat("yaml"), false},
		{UserDataFormat("xlsx"), false},
	}
	for _, tt := range cases {
		if got := tt.format.Valid(); got != tt.want {
			t.Errorf("UserDataFormat(%q).Valid() = %v, want %v", tt.format, got, tt.want)
		}
	}
}
