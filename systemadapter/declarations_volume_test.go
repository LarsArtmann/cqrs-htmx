package systemadapter

import "testing"

// TestDeclarativeProjections_CarryVolumeHints pins the Volume provenance:
// every metaengine query declaration must carry a positive Volume hint
// because the cost-based planner ranks engines by expected collection size —
// a missing hint silently degrades placement to the memory engine even for
// large collections (audit log, user scan).
func TestDeclarativeProjections_CarryVolumeHints(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		queryName string
		volume    int64
		want      int64
	}{
		{"tenantLookup", "tenant_by_id", tenantLookup().Config.Volume, 100},
		{"tenantScan", "tenants", tenantScan().Config.Volume, 100},
		{"botLookup", "bot_by_id", botLookup().Config.Volume, 1_000},
		{"botScan", "bots", botScan().Config.Volume, 1_000},
		{"botTokenScan", "bot_tokens", botTokenScan().Config.Volume, 1_000},
		{"membershipLookup", "membership_by_id", membershipLookup().Config.Volume, 10_000},
		{"membershipScan", "memberships", membershipScan().Config.Volume, 10_000},
		{"userLookup", "user_by_id", userLookup().Config.Volume, 100_000},
		{"userScan", "users", userScan().Config.Volume, 100_000},
		{"authzPolicyLookup", "authz_policy_by_id", authzPolicyLookup().Config.Volume, 10_000},
		{"authzPolicyScan", "authz_policies", authzPolicyScan().Config.Volume, 10_000},
		{"auditLogScan", "audit_log", auditLogScan().Config.Volume, 1_000_000},
	} {
		if tc.volume != tc.want {
			t.Errorf("%s (%s): Volume = %d, want %d", tc.name, tc.queryName, tc.volume, tc.want)
		}
	}

	// The wrap into system.ProjectionDeclaration must not drop any query:
	// when a constructor is added, extend the table above.
	if got := len(DeclarativeProjections()); got != 12 {
		t.Errorf("DeclarativeProjections() = %d declarations, want 12 (new query? extend the Volume table)", got)
	}
}
