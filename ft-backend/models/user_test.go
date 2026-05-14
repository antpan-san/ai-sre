package models

import "testing"

func TestUserRoleHelpers(t *testing.T) {
	for _, role := range []string{RoleSuperAdmin, RoleAdmin, RoleUser} {
		if !IsValidUserRole(role) {
			t.Fatalf("expected %s to be valid", role)
		}
	}
	if IsValidUserRole("operator") {
		t.Fatalf("operator must not be assignable as a login role")
	}
	if !IsAdminRole(RoleSuperAdmin) || !IsAdminRole(RoleAdmin) {
		t.Fatalf("admin roles should include admin and super_admin")
	}
	if IsSuperAdminRole(RoleAdmin) {
		t.Fatalf("admin must not be treated as super_admin")
	}
}

func TestKnownFeatureKeys(t *testing.T) {
	for _, k := range []string{
		FeatureKeyAdvanced,
		FeatureKeyK8sOps,
		FeatureKeyServiceOps,
		FeatureKeyInfraOps,
	} {
		if !IsKnownFeatureKey(k) {
			t.Fatalf("%s should be known", k)
		}
	}
	if IsKnownFeatureKey("feature.anything") {
		t.Fatalf("unknown feature keys must be rejected")
	}
}
