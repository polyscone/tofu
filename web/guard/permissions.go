package guard

const (
	viewRoles   = "account:roles:view"
	createRoles = "account:roles:create"
	updateRoles = "account:roles:update"
	deleteRoles = "account:roles:delete"

	viewUsers        = "account:users:view"
	inviteUsers      = "account:users:invite"
	activateUsers    = "account:users:activate"
	changeRoles      = "account:users:update:roles"
	reviewTOTPResets = "account:users:review_totp_reset"
	suspendUsers     = "account:users:suspend"
	unsuspendUsers   = "account:users:unsuspend"

	viewConfig   = "system:config:view"
	updateConfig = "system:config:update"
	viewMetrics  = "system:metrics:view"
)

var PermissionGroups = []PermissionGroup{
	{
		Name: "Account roles",
		Permissions: []Permission{
			{DisplayName: "View roles", Name: viewRoles},
			{DisplayName: "Create roles", Name: createRoles},
			{DisplayName: "Edit roles", Name: updateRoles},
			{DisplayName: "Delete roles", Name: deleteRoles},
		},
	},
	{
		Name: "Account users",
		Permissions: []Permission{
			{DisplayName: "View users", Name: viewUsers},
			{DisplayName: "Invite users", Name: inviteUsers},
			{DisplayName: "Activate users", Name: activateUsers},
			{DisplayName: "Change user roles", Name: changeRoles},
			{DisplayName: "Review 2FA resets", Name: reviewTOTPResets},
			{DisplayName: "Suspend users", Name: suspendUsers},
			{DisplayName: "Unsuspend users", Name: unsuspendUsers},
		},
	},
	{
		Name: "System",
		Permissions: []Permission{
			{DisplayName: "View system config", Name: viewConfig},
			{DisplayName: "Edit system config", Name: updateConfig},
			{DisplayName: "View system metrics", Name: viewMetrics},
		},
	},
}

type Permission struct {
	DisplayName string
	Name        string
}

type PermissionGroup struct {
	Name        string
	Permissions []Permission
}
