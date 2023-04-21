package sqlite

import (
	"context"

	"github.com/polyscone/tofu/internal/pkg/aesgcm"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/repo/sqlite"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port/account/internal/domain"
)

func (r *UserRepo) findRecoveryCodesByUserID(ctx context.Context, db sqlite.Querier, userID uuid.V4) ([]domain.RecoveryCode, error) {
	var recoveryCodes []domain.RecoveryCode

	stmt, args := `
		SELECT code
		FROM account__recovery_codes
		WHERE user_id = :user_id;
	`, sqlite.Args{"user_id": userID}
	rows, err := db.Query(ctx, stmt, args)
	if err != nil {
		return recoveryCodes, errors.Tracef(err)
	}
	for rows.Next() {
		var encrypted []byte

		if err := rows.Scan(&encrypted); err != nil {
			return recoveryCodes, errors.Tracef(err)
		}

		decrypted, err := aesgcm.Decrypt(r.secret, encrypted)
		if err != nil {
			return recoveryCodes, errors.Tracef(err)
		}

		recoveryCode, err := domain.NewRecoveryCode(string(decrypted))
		if err != nil {
			return recoveryCodes, errors.Tracef(err)
		}

		recoveryCodes = append(recoveryCodes, recoveryCode)
	}

	return recoveryCodes, errors.Tracef(rows.Err())
}
