package account

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app"
)

type DeleteRoleGuard interface {
	CanDeleteRoles() bool
}

type DeleteRoleData struct {
	RoleID int
}

func (s *Service) DeleteRoleValidate(guard DeleteRoleGuard, roleID int) (DeleteRoleData, error) {
	var data DeleteRoleData

	if !guard.CanDeleteRoles() {
		return data, app.ErrForbidden
	}

	data.RoleID = roleID

	return data, nil
}

func (s *Service) DeleteRole(ctx context.Context, guard DeleteRoleGuard, roleID int) (*Role, error) {
	data, err := s.DeleteRoleValidate(guard, roleID)
	if err != nil {
		return nil, err
	}

	role, err := s.repo.FindRoleByID(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("find role by id: %w", err)
	}

	if err := s.repo.RemoveRole(ctx, data.RoleID); err != nil {
		return nil, fmt.Errorf("remove role: %w", err)
	}

	return role, nil
}
