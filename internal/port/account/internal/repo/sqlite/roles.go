package sqlite

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port/account/internal/domain"
)

func (r *UserRepo) findRolesByUserID(ctx context.Context, db sqlite.Querier, userID uuid.V4) ([]domain.Role, error) {
	var roles []domain.Role

	stmt, args := `
		SELECT id, name
		FROM account__roles AS r
		INNER JOIN account__user_roles AS ur ON r.id = ur.role_id
		WHERE ur.user_id = :user_id;
	`, sqlite.Args{"user_id": userID}
	rows, err := db.Query(ctx, stmt, args)
	if err != nil {
		return roles, errors.Tracef(err)
	}
	for rows.Next() {
		var roleID uuid.V4
		var roleName string

		if err := rows.Scan(&roleID, &roleName); err != nil {
			return roles, errors.Tracef(err)
		}

		var permissions []domain.Permission

		stmt, args := `
			SELECT id
			FROM account__permissions AS p
			INNER JOIN account__role_permissions AS rp ON p.id = rp.permission_id
			WHERE rp.role_id = :role_id;
		`, sqlite.Args{"role_id": roleID}
		rows, err := db.Query(ctx, stmt, args)
		if err != nil {
			return roles, errors.Tracef(err)
		}
		for rows.Next() {
			var permissionID domain.Permission

			if err := rows.Scan(&permissionID); err != nil {
				return roles, errors.Tracef(err)
			}

			permissions = append(permissions, permissionID)
		}
		if err := rows.Err(); err != nil {
			return roles, errors.Tracef(err)
		}

		roles = append(roles, domain.NewRole(roleName, permissions...))
	}

	return roles, errors.Tracef(rows.Err())
}
