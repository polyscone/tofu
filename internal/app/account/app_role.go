package account

type Role struct {
	ID          int
	Name        string
	Permissions []*Permission
}

type RoleFilter struct {
	UserID *int
}
