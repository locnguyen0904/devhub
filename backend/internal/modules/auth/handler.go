package auth

import (
	"context"
	"log/slog"
	"net/http"
	"net/netip"

	"github.com/locnguyen0904/devhub/backend/internal/platform/httpx"
)

// Handler translates HTTP to the service. The two GitHub endpoints are plain
// net/http handlers because they are browser redirects that set cookies, which
// do not fit huma's typed request/response model; the JSON endpoints are huma
// operations so they appear in the OpenAPI spec.
type Handler struct {
	svc         *service
	log         *slog.Logger
	frontendURL string
	secure      bool
}

func newHandler(svc *service, log *slog.Logger, frontendURL string, secure bool) *Handler {
	return &Handler{svc: svc, log: log, frontendURL: frontendURL, secure: secure}
}

// redirectToGitHub sends the browser to GitHub's authorization page.
func (h *Handler) redirectToGitHub(w http.ResponseWriter, r *http.Request) {
	url, err := h.svc.AuthorizeURL(r.Context())
	if err != nil {
		h.log.ErrorContext(r.Context(), "build authorize url", slog.String("error", err.Error()))
		http.Error(w, "could not start login", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, url, http.StatusFound)
}

// callback handles GitHub's redirect back: exchange the code, set the refresh
// cookie, and bounce the browser to the frontend. On failure it redirects to
// the frontend with a flag rather than showing a raw error page.
func (h *Handler) callback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("error") != "" {
		h.redirectFailure(w, r)
		return
	}

	tokens, err := h.svc.Complete(r.Context(), q.Get("code"), q.Get("state"), sessionMeta(r))
	if err != nil {
		h.log.ErrorContext(r.Context(), "oauth callback", slog.String("error", err.Error()))
		h.redirectFailure(w, r)
		return
	}

	http.SetCookie(w, h.refreshCookie(tokens.Refresh, tokens.RefreshMaxAge))
	http.Redirect(w, r, h.frontendURL, http.StatusFound)
}

func (h *Handler) redirectFailure(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, h.frontendURL+"?auth_error=1", http.StatusFound)
}

// refresh rotates the session and returns a fresh access token.
func (h *Handler) refresh(ctx context.Context, in *RefreshInput) (*SessionOutput, error) {
	if in.Cookie.Value == "" {
		return nil, httpx.ToHuma(httpx.New(httpx.CodeUnauthenticated, "No active session", nil))
	}

	tokens, err := h.svc.Refresh(ctx, in.Cookie.Value, SessionMeta{})
	if err != nil {
		return nil, httpx.ToHuma(err)
	}

	out := &SessionOutput{SetCookie: *h.refreshCookie(tokens.Refresh, tokens.RefreshMaxAge)}
	out.Body = SessionResponse{
		AccessToken: tokens.Access,
		TokenType:   "Bearer",
		ExpiresIn:   tokens.AccessExpiresIn,
		User:        toUserView(tokens),
	}
	return out, nil
}

// logout revokes the presented refresh token and clears the cookie.
func (h *Handler) logout(ctx context.Context, in *LogoutInput) (*LogoutOutput, error) {
	if in.Cookie.Value != "" {
		if err := h.svc.Logout(ctx, in.Cookie.Value); err != nil {
			return nil, httpx.ToHuma(err)
		}
	}
	return &LogoutOutput{SetCookie: *h.clearedCookie()}, nil
}

// logoutAll revokes every session for the authenticated user.
func (h *Handler) logoutAll(ctx context.Context, _ *struct{}) (*MessageOutput, error) {
	identity, err := RequireIdentity(ctx)
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	if err := h.svc.LogoutAll(ctx, identity.UserID); err != nil {
		return nil, httpx.ToHuma(err)
	}
	out := &MessageOutput{}
	out.Body.Status = "ok"
	return out, nil
}

func (h *Handler) refreshCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     refreshCookieName,
		Value:    value,
		Path:     refreshCookiePath,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteStrictMode,
	}
}

func (h *Handler) clearedCookie() *http.Cookie {
	c := h.refreshCookie("", -1)
	return c
}

// sessionMeta captures who and where a session was created from, stored with the
// refresh token so a user can later review and revoke individual sessions.
func sessionMeta(r *http.Request) SessionMeta {
	var meta SessionMeta
	if ua := r.UserAgent(); ua != "" {
		meta.UserAgent = &ua
	}
	if addr, err := netip.ParseAddrPort(r.RemoteAddr); err == nil {
		ip := addr.Addr()
		meta.IP = &ip
	}
	return meta
}
