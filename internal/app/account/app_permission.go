package account

type Permission struct {
	ID   string
	Name string
}

type PermissionFilter struct {
	RoleID *int
}
