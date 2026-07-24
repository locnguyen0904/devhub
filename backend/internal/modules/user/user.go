// Package user owns the users table and everything about a user's identity.
// Other modules reach it through the Service interface, never the repository.
package user

import (
	"time"

	"github.com/google/uuid"
)

// User is the domain model. It is mapped from the sqlc row in the repository so
// that generated types never leak past this package.
type User struct {
	ID             uuid.UUID
	Username       string
	Email          *string
	DisplayName    string
	AvatarURL      *string
	GitHubUsername *string
	Role           string
	CreatedAt      time.Time
}

// Brief is the small slice of a user other modules embed when displaying
// authorship — a post card needs the name and avatar, nothing more.
type Brief struct {
	ID          uuid.UUID
	Username    string
	DisplayName string
	AvatarURL   *string
}

// GitHubIdentity is what the auth module hands over after talking to GitHub.
// user turns it into a User; auth never touches the users table itself.
type GitHubIdentity struct {
	GitHubID  string // GitHub's numeric account id, stable across renames
	Login     string // GitHub username, used to seed our username
	Name      string // display name; may be empty
	Email     *string
	AvatarURL *string
}
