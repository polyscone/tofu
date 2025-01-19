package account

import (
	"context"
	"fmt"
)

type DenyTOTPResetRequestData struct {
	UserID int
}

func (s *Service) DenyTOTPResetRequestValidate(userID int) (DenyTOTPResetRequestData, error) {
	var data DenyTOTPResetRequestData

	data.UserID = userID

	return data, nil
}

func (s *Service) DenyTOTPResetRequest(ctx context.Context, userID int) (*User, error) {
	data, err := s.DenyTOTPResetRequestValidate(userID)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, data.UserID)
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
