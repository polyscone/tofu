package account

type Role struct {
	ID          int
	Name        string
	Description string
	Permissions []string
}

type RoleFilter struct {
	ID     *int
	UserID *int
	Search *string

	SortTopID int

	Limit  int
	Offset int
}

func NewRole(name RoleName, description RoleDesc, permissions []Permission) *Role {
	role := Role{
		Name:        name.String(),
		Description: description.String(),
	}

	if n := len(permissions); n != 0 {
		role.Permissions = make([]string, n)

		for i, permission := range permissions {
			role.Permissions[i] = permission.String()
		}
	}

	return &role
}
