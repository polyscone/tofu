package account

import "github.com/polyscone/tofu/internal/pkg/aggregate"

type Role struct {
	aggregate.Root

	ID          int
	Name        string
	Description string
	Permissions []string
}

type RoleFilter struct {
	ID     *int
	UserID *int
	Name   *string
	Search *string

	SortTopID int
	Sorts     []string

	Limit  int
	Offset int
}

func NewRole(name RoleName, description RoleDesc, permissions []Permission) *Role {
	role := Role{
		Name:        name.String(),
		Description: description.String(),
	}

	if permissions != nil {
		role.Permissions = make([]string, len(permissions))

		for i, permission := range permissions {
			role.Permissions[i] = permission.String()
		}
	}

	return &role
}
