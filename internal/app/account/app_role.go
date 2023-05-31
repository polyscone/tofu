package account

type Role struct {
	ID          int
	Name        string
	Permissions []*Permission
}

type RoleFilter struct {
	ID     *int
	UserID *int
	Search *string

	Limit  int
	Offset int
}

func NewRole(name string) *Role {
	return &Role{Name: name}
}
