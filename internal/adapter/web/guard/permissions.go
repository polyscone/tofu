package guard

const (
	viewRoles   = "account:roles:view"
	createRoles = "account:roles:create"
	editRoles   = "account:roles:edit"
	deleteRoles = "account:roles:delete"

	viewUsers   = "account:users:view"
	editUsers   = "account:users:edit"
	changeRoles = "account:users:edit:roles"
)

var PermissionGroups = []PermissionGroup{
	{
		Name: "Account roles",
		Permissions: []Permission{
			{DisplayName: "View roles", Name: viewRoles},
			{DisplayName: "Create roles", Name: createRoles},
			{DisplayName: "Edit roles", Name: editRoles},
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
}

type Permission struct {
	DisplayName string
	Name        string
}

type PermissionGroup struct {
	Name        string
	Permissions []Permission
}
