package account

import (
	"context"
	"fmt"
)

type ApproveTOTPResetRequestData struct {
	UserID int
}

func (s *Service) ApproveTOTPResetRequestValidate(userID int) (ApproveTOTPResetRequestData, error) {
	var data ApproveTOTPResetRequestData

	data.UserID = userID

	return data, nil
}

func (s *Service) ApproveTOTPResetRequest(ctx context.Context, userID int) (*User, error) {
	data, err := s.ApproveTOTPResetRequestValidate(userID)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(ctx, data.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	if err := user.ApproveTOTPResetRequest(); err != nil {
		return nil, err
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	s.broker.Flush(ctx, &user.Events)

	return user, nil
}
