package ports

import (
	"context"
	"time"

	"tutor-platform/services/core/internal/auth"
)

type IdentityProvider interface {
	Authenticate(ctx context.Context, credential string) (auth.Principal, error)
}

type ObjectStorage interface {
	PutPrivate(ctx context.Context, objectName string, contentType string, body []byte) error
	SignedDownloadURL(ctx context.Context, objectName string, ttl time.Duration) (string, error)
	Delete(ctx context.Context, objectName string) error
}

type Messenger interface {
	CreateRoom(ctx context.Context, relationID string, participantIDs []string) (string, error)
	SendSystemMessage(ctx context.Context, roomID string, body string) error
}

type VideoProvider interface {
	CreateRoom(ctx context.Context, lessonID string, participantIDs []string, startsAt time.Time, endsAt time.Time) (string, error)
}
