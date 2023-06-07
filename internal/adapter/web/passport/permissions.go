package passport

var PermissionGroups = []PermissionGroup{
	{
		Name: "Account roles",
		Permissions: []Permission{
			{Name: "account:roles:view", DisplayName: "View roles"},
			{Name: "account:roles:create", DisplayName: "Create roles"},
			{Name: "account:roles:edit", DisplayName: "Edit roles"},
			{Name: "account:roles:delete", DisplayName: "Delete roles"},
		},
	},
	{
		Name: "Account users",
		Permissions: []Permission{
			{Name: "account:users:view", DisplayName: "View users"},
			{Name: "account:users:edit", DisplayName: "Edit users"},
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
