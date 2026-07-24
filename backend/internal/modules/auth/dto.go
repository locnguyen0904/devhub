package auth

import "net/http"

// refreshCookieName is the cookie carrying the refresh token. Its Path is
// scoped to /api/v1/auth so the browser only sends it to auth endpoints (refresh
// and logout), keeping it off every other API call.
const (
	refreshCookieName = "devhub_refresh"
	refreshCookiePath = "/api/v1/auth"
)

// SessionResponse is the JSON body returned by refresh. It carries the user so
// the frontend needs no second call to render who is signed in.
type SessionResponse struct {
	AccessToken string   `json:"access_token" doc:"Short-lived bearer token, kept in memory by the client"`
	TokenType   string   `json:"token_type" example:"Bearer"`
	ExpiresIn   int      `json:"expires_in" doc:"Seconds until the access token expires"`
	User        UserView `json:"user"`
}

// UserView is the public shape of the signed-in user.
type UserView struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

// RefreshInput reads the refresh token from the cookie the browser sends.
type RefreshInput struct {
	Cookie http.Cookie `cookie:"devhub_refresh"`
}

// SessionOutput returns the new access token and rotates the refresh cookie.
type SessionOutput struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
	Body      SessionResponse
}

// LogoutInput reads the refresh cookie so the matching token can be revoked.
type LogoutInput struct {
	Cookie http.Cookie `cookie:"devhub_refresh"`
}

// LogoutOutput clears the refresh cookie.
type LogoutOutput struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
}

// MessageOutput is a minimal acknowledgement body.
type MessageOutput struct {
	Body struct {
		Status string `json:"status" example:"ok"`
	}
}

func toUserView(t Tokens) UserView {
	return UserView{
		ID:          t.User.ID.String(),
		Username:    t.User.Username,
		DisplayName: t.User.DisplayName,
		AvatarURL:   t.User.AvatarURL,
	}
}
