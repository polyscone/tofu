package account

import (
	"context"
	"fmt"
)

type DenyTOTPResetRequestInput struct {
	UserID int
}

func (s *Service) DenyTOTPResetRequestValidate(userID int) (DenyTOTPResetRequestInput, error) {
	var input DenyTOTPResetRequestInput

	input.UserID = userID

	return input, nil
}

func (s *Service) DenyTOTPResetRequest(ctx context.Context, userID int) (*User, error) {
	input, err := s.DenyTOTPResetRequestValidate(userID)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	if err := user.DenyTOTPResetRequest(); err != nil {
		return nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(ctx, &user.Events)

	return user, nil
}
