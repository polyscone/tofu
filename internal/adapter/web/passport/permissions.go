package passport

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
			{Name: viewRoles, DisplayName: "View roles"},
			{Name: createRoles, DisplayName: "Create roles"},
			{Name: editRoles, DisplayName: "Edit roles"},
			{Name: deleteRoles, DisplayName: "Delete roles"},
		},
	},
	{
		Name: "Account users",
		Permissions: []Permission{
			{Name: viewUsers, DisplayName: "View users"},
			{Name: editUsers, DisplayName: "Edit users"},
			{Name: changeRoles, DisplayName: "Change user roles"},
		},
	},
}

type Permission struct {
	Name        string
	DisplayName string
}

type PermissionGroup struct {
	Name        string
	Permissions []Permission
}
