package account

import "github.com/polyscone/tofu/internal/pkg/valobj/text"

type Role struct {
	ID          int
	Name        string
	Permissions []*Permission
}

type RoleFilter struct {
	ID     *int
	UserID *int
	Search *string

	SortTopID int

	Limit  int
	Offset int
}

func NewRole(name text.Name) *Role {
	return &Role{Name: name.String()}
}
