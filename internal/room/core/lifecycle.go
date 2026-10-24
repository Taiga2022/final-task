package core

import "context"

// MembershipLifecycle serializes persistent membership changes with live room state.
type MembershipLifecycle interface {
	ChangeMembership(ctx context.Context, roomID, userID string, joining bool, change func() error) error
}
