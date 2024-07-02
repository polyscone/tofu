package account

import (
	"context"
	"fmt"
)

func (s *Service) ApproveTOTPResetRequest(ctx context.Context, userID int) (*User, error) {
	var input struct {
		userID int
	}
	{
		input.userID = userID
	}

	user, err := s.repo.FindUserByID(ctx, input.userID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	if err := user.ApproveTOTPResetRequest(); err != nil {
		return nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(&user.Events)

	return user, nil
}
