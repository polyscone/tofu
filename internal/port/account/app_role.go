package account

type Role struct {
	Name        string
	Permissions []Permission
}

func NewRole(name string, permissions ...Permission) Role {
	return Role{
		Name:        name,
		Permissions: permissions,
	}
}
