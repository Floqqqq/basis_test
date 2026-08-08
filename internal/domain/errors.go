package domain

import "errors"

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrTeamMemberExists = errors.New("user is already team member")
)
