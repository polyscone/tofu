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
			{DisplayName: "Create roles", Name: createRoles, Implies: []string{viewRoles}},
			{DisplayName: "Edit roles", Name: editRoles, Implies: []string{viewRoles}},
			{DisplayName: "Delete roles", Name: deleteRoles, Implies: []string{viewRoles}},
		},
	},
	{
		Name: "Account users",
		Permissions: []Permission{
			{DisplayName: "View users", Name: viewUsers},
			{DisplayName: "Edit users", Name: editUsers, Implies: []string{viewUsers}},
			{DisplayName: "Change user roles", Name: changeRoles, Implies: []string{viewUsers, editUsers}},
		},
	},
}

type Permission struct {
	DisplayName string
	Name        string
	Implies     []string
}

type PermissionGroup struct {
	Name        string
	Permissions []Permission
}

func ExpandPermissions(names []string) []string {
	expanded := make(map[string]struct{}, len(names))

	for _, name := range names {
		expanded[name] = struct{}{}

		for _, group := range PermissionGroups {
			for _, permission := range group.Permissions {
				if name != permission.Name {
					continue
				}

				for _, implied := range permission.Implies {
					expanded[implied] = struct{}{}
				}
			}
		}
	}

	permissions := make([]string, 0, len(expanded))
	for name := range expanded {
		permissions = append(permissions, name)
	}

	return permissions
}
