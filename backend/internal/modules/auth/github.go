package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/locnguyen0904/devhub/backend/internal/modules/user"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// errGitHubStatus is returned when the GitHub API answers with a non-200 status.
var errGitHubStatus = errors.New("github api returned non-200 status")

// githubClient wraps the OAuth exchange and profile fetch. It is the only thing
// in the module that talks to GitHub over the network.
type githubClient struct {
	oauth *oauth2.Config
	http  *http.Client
}

func newGitHubClient(clientID, clientSecret, redirectURL string) *githubClient {
	return &githubClient{
		oauth: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"read:user", "user:email"},
			Endpoint:     github.Endpoint,
		},
		http: &http.Client{},
	}
}

// authCodeURL builds the GitHub authorization URL the browser is sent to.
func (g *githubClient) authCodeURL(state string) string {
	return g.oauth.AuthCodeURL(state)
}

// exchange trades the callback code for the identity. The GitHub access token
// is used here to read the profile and then discarded — Phase 1 does not store
// it. GitHub sync (Phase 7) is what will keep and encrypt it.
func (g *githubClient) exchange(ctx context.Context, code string) (user.GitHubIdentity, error) {
	tok, err := g.oauth.Exchange(ctx, code)
	if err != nil {
		return user.GitHubIdentity{}, fmt.Errorf("exchange oauth code: %w", err)
	}

	client := g.oauth.Client(ctx, tok)

	profile, err := g.fetchProfile(ctx, client)
	if err != nil {
		return user.GitHubIdentity{}, err
	}

	// GitHub omits the email when the user keeps it private; fetch it separately.
	if profile.Email == nil {
		if email := g.fetchPrimaryEmail(ctx, client); email != "" {
			profile.Email = &email
		}
	}
	return profile, nil
}

type githubProfile struct {
	ID        int64   `json:"id"`
	Login     string  `json:"login"`
	Name      string  `json:"name"`
	Email     *string `json:"email"`
	AvatarURL *string `json:"avatar_url"`
}

func (g *githubClient) fetchProfile(ctx context.Context, client *http.Client) (user.GitHubIdentity, error) {
	var p githubProfile
	if err := getJSON(ctx, client, "https://api.github.com/user", &p); err != nil {
		return user.GitHubIdentity{}, fmt.Errorf("fetch github profile: %w", err)
	}
	return user.GitHubIdentity{
		GitHubID:  strconv.FormatInt(p.ID, 10),
		Login:     p.Login,
		Name:      p.Name,
		Email:     p.Email,
		AvatarURL: p.AvatarURL,
	}, nil
}

func (g *githubClient) fetchPrimaryEmail(ctx context.Context, client *http.Client) string {
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	// A missing email is not fatal: the user row allows a null email, so swallow
	// the error and return empty rather than failing the whole login.
	if err := getJSON(ctx, client, "https://api.github.com/user/emails", &emails); err != nil {
		return ""
	}
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email
		}
	}
	return ""
}

func getJSON(ctx context.Context, client *http.Client, url string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %d", errGitHubStatus, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
