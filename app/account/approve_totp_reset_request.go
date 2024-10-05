package account

import (
	"context"
	"fmt"
)

type ApproveTOTPResetRequestInput struct {
	UserID int
}

func (s *Service) ApproveTOTPResetRequestValidate(userID int) (ApproveTOTPResetRequestInput, error) {
	var input ApproveTOTPResetRequestInput

	input.UserID = userID

	return input, nil
}

func (s *Service) ApproveTOTPResetRequest(ctx context.Context, userID int) (*User, error) {
	input, err := s.ApproveTOTPResetRequestValidate(userID)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, input.UserID)
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
