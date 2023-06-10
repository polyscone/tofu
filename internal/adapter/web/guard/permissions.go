package guard

const (
	viewRoles   = "account:roles:view"
	createRoles = "account:roles:create"
	updateRoles = "account:roles:update"
	deleteRoles = "account:roles:delete"

	viewUsers   = "account:users:view"
	editUsers   = "account:users:update"
	changeRoles = "account:users:update:roles"

	viewConfig   = "web:config:view"
	updateConfig = "web:config:update"
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
			{DisplayName: "Edit users", Name: editUsers},
			{DisplayName: "Change user roles", Name: changeRoles},
		},
	},
	{
		Name: "System",
		Permissions: []Permission{
			{DisplayName: "View system config", Name: viewConfig},
			{DisplayName: "Edit system config", Name: updateConfig},
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
