package domain

import "errors"

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrTeamMemberExists    = errors.New("user is already team member")
	ErrLeaveRequestExists  = errors.New("leave request already exists")
	ErrLeaveRequestHandled = errors.New("leave request already handled")
	ErrCannotResolveOwn    = errors.New("cannot resolve own leave request")
)
