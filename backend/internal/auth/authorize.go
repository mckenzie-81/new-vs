package auth

import (
	"errors"
	"net/http"
	"slices"

	"backend/internal/httpx"
)

var (
	ErrUnauthorized = errors.New("not authenticated")
	ErrForbidden    = errors.New("forbidden")
)

// Authorize is the one place every protected endpoint, in any package,
// checks who's making the request. It validates the session cookie and,
// when allowedRoles is non-empty, checks the session's role is one of
// them. The cookie is cleared only when the session itself was invalid -
// not when it was valid but for the wrong role, since a legitimate user
// attempting an action they can't do shouldn't be logged out over it.
func (s *Service) Authorize(w http.ResponseWriter, r *http.Request, allowedRoles ...string) (*User, error) {
	rawToken := readSessionCookie(r)
	user, err := s.ValidateSession(r.Context(), rawToken, httpx.ClientIP(r), r.Header.Get("User-Agent"))
	if err != nil {
		clearSessionCookie(w, s.cookieSecure)
		return nil, ErrUnauthorized
	}

	if len(allowedRoles) > 0 && !slices.Contains(allowedRoles, user.Role) {
		return nil, ErrForbidden
	}

	return user, nil
}
