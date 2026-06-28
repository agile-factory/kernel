package permission

import "github.com/agile-factory/kernel/platform/membership/domain"

// rolePermissions maps each MemberRole to the set of Permissions it holds.
// This is the v1 authoritative source; update here when a new permission
// constant is added above. v2 will replace this map with a DB-backed lookup.
var rolePermissions = map[domain.MemberRole]map[Permission]struct{}{
	domain.MEMBER_ROLE_OWNER: permSet(
		TenantView, TenantUpdate, TenantDelete,
		MemberInvite, MemberRevoke, MemberList,
		AuditView,
		UserProfileView, UserProfileUpdateOwn,
		StorageUpload, StorageDelete,
	),
	domain.MEMBER_ROLE_ADMIN: permSet(
		TenantView, TenantUpdate, // no TenantDelete — only OWNER can delete a tenant
		MemberInvite, MemberRevoke, MemberList,
		AuditView,
		UserProfileView, UserProfileUpdateOwn,
		StorageUpload, StorageDelete,
	),
	domain.MEMBER_ROLE_AGENT: permSet(
		TenantView,
		MemberList,
		UserProfileView, UserProfileUpdateOwn,
		StorageUpload,
	),
	domain.MEMBER_ROLE_VIEWER: permSet(
		TenantView,
		MemberList,
		AuditView,
		UserProfileView,
	),
}

func permSet(perms ...Permission) map[Permission]struct{} {
	s := make(map[Permission]struct{}, len(perms))
	for _, p := range perms {
		s[p] = struct{}{}
	}
	return s
}

// roleHasPermission reports whether role grants perm.
func roleHasPermission(role domain.MemberRole, perm Permission) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	_, has := perms[perm]
	return has
}
