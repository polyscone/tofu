package domain

var (
	CreateProjects = Permission("CreateProjects")
)

type Permission string

func (p Permission) String() string {
	return string(p)
}
