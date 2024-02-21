package account

import "github.com/polyscone/tofu/internal/pkg/aggregate"

type Role struct {
	aggregate.Root

	ID          string
	Name        string
	Description string
	Permissions []string
}

type RoleFilter struct {
	ID     *string
	UserID *string
	Name   *string
	Search *string

	SortTopID string
	Sorts     []string

	Limit  int
	Offset int
}

func NewRole(id ID, name RoleName, description RoleDesc, permissions []Permission) *Role {
	role := Role{
		ID:          id.String(),
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
