package guard

import "github.com/polyscone/tofu/internal/i18n"

const (
	viewRoles   = "account.roles.view"
	createRoles = "account.roles.create"
	updateRoles = "account.roles.update"
	deleteRoles = "account.roles.delete"

	viewUsers           = "account.users.view"
	inviteUsers         = "account.users.invite"
	activateUsers       = "account.users.activate"
	canImpersonateUsers = "account.users.impersonate"
	changeRoles         = "account.users.update.roles"
	reviewTOTPResets    = "account.users.review_totp_reset"
	suspendUsers        = "account.users.suspend"
	unsuspendUsers      = "account.users.unsuspend"

	viewConfig   = "system.config.view"
	updateConfig = "system.config.update"
	backup       = "system.recovery.backup"
	restore      = "system.recovery.restore"
	viewMetrics  = "system.metrics.view"
)

var PermissionGroups = []PermissionGroup{
	{
		Name: i18n.M("web.guard.permission.account.role.group_name"),
		Permissions: []Permission{
			{DisplayName: i18n.M("web.guard.permission.account.role.view"), Name: viewRoles},
			{DisplayName: i18n.M("web.guard.permission.account.role.create"), Name: createRoles},
			{DisplayName: i18n.M("web.guard.permission.account.role.edit"), Name: updateRoles},
			{DisplayName: i18n.M("web.guard.permission.account.role.delete"), Name: deleteRoles},
		},
	},
	{
		Name: i18n.M("web.guard.permission.account.user.group_name"),
		Permissions: []Permission{
			{DisplayName: i18n.M("web.guard.permission.account.user.view"), Name: viewUsers},
			{DisplayName: i18n.M("web.guard.permission.account.user.invite"), Name: inviteUsers},
			{DisplayName: i18n.M("web.guard.permission.account.user.activate"), Name: activateUsers},
			{DisplayName: i18n.M("web.guard.permission.account.user.impersonate"), Name: canImpersonateUsers},
			{DisplayName: i18n.M("web.guard.permission.account.user.change_roles"), Name: changeRoles},
			{DisplayName: i18n.M("web.guard.permission.account.user.review_totp_reset_requests"), Name: reviewTOTPResets},
			{DisplayName: i18n.M("web.guard.permission.account.user.suspend"), Name: suspendUsers},
			{DisplayName: i18n.M("web.guard.permission.account.user.unsuspend"), Name: unsuspendUsers},
		},
	},
	{
		Name: i18n.M("web.guard.permission.system.group_name"),
		Permissions: []Permission{
			{DisplayName: i18n.M("web.guard.permission.system.config.view"), Name: viewConfig},
			{DisplayName: i18n.M("web.guard.permission.system.config.edit"), Name: updateConfig},
			{DisplayName: i18n.M("web.guard.permission.system.recovery.backup"), Name: backup},
			{DisplayName: i18n.M("web.guard.permission.system.recovery.restore"), Name: restore},
			{DisplayName: i18n.M("web.guard.permission.system.metrics.view"), Name: viewMetrics},
		},
	},
}

type Permission struct {
	DisplayName i18n.Message
	Name        string
}

type PermissionGroup struct {
	Name        i18n.Message
	Permissions []Permission
}
