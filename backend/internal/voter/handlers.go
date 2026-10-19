package voter

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"backend/internal/audit"
	"backend/internal/auth"
	"backend/internal/election"
	"backend/internal/httpx"

	"github.com/jackc/pgx/v5/pgxpool"
)

// dashboardRoles mirrors the identical private list in internal/election -
// small enough to duplicate rather than export just for this one call site.
var dashboardRoles = []string{"SYSTEMADMIN", "ADMIN", "APPROVER"}

type Handler struct {
	service         *Service
	electionService *election.Service
	authService     *auth.Service
	pool            *pgxpool.Pool
	appBaseURL      string
	cookieSecure    bool
}

func NewHandler(service *Service, electionService *election.Service, authService *auth.Service, pool *pgxpool.Pool, appBaseURL string, cookieSecure bool) *Handler {
	return &Handler{
		service:         service,
		electionService: electionService,
		authService:     authService,
		pool:            pool,
		appBaseURL:      appBaseURL,
		cookieSecure:    cookieSecure,
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/elections/{electionId}/voter-credentials/issue", h.issueCredentials)
	mux.HandleFunc("GET /api/elections/{id}/results", h.getResults)
	mux.HandleFunc("GET /api/vote/{token}", h.getTokenStatus)
	mux.HandleFunc("POST /api/vote/{token}/authenticate", h.authenticate)
	mux.HandleFunc("GET /api/vote/ballot", h.getBallot)
	mux.HandleFunc("POST /api/vote/cast", h.castVote)
}

type inviteResponse struct {
	Email string `json:"email"`
	Link  string `json:"link"`
}

type issueResponse struct {
	Eligible      int              `json:"eligible"`
	Issued        int              `json:"issued"`
	AlreadyIssued int              `json:"alreadyIssued"`
	Invites       []inviteResponse `json:"invites"`
}

// issueCredentials is an admin action, so - unlike the public voter routes
// below - invite links are never logged (see the package doc for why):
// they're returned here, in the authenticated admin's own response, which
// is also simply more convenient for testing than grepping stdout.
func (h *Handler) issueCredentials(w http.ResponseWriter, r *http.Request) {
	electionID := r.PathValue("electionId")
	actor, err := election.AuthorizeBallotEditor(h.authService, h.electionService, w, r, electionID)
	if err != nil {
		election.WriteBallotAuthError(w, err)
		return
	}

	result, err := h.service.IssueAndInviteCredentials(r.Context(), electionID)
	switch {
	case err == nil:
		resp := issueResponse{
			Eligible:      result.Eligible,
			Issued:        len(result.Issued),
			AlreadyIssued: result.AlreadyIssued,
			Invites:       make([]inviteResponse, 0, len(result.Issued)),
		}
		for _, invite := range result.Issued {
			resp.Invites = append(resp.Invites, inviteResponse{
				Email: invite.Email,
				Link:  h.appBaseURL + "/vote/" + invite.RawToken,
			})
		}
		h.logAudit(r.Context(), audit.Entry{
			ActorUserID: &actor.ID,
			Action:      "VOTER_CREDENTIALS_ISSUED",
			Metadata:    map[string]any{"election_id": electionID, "issued": resp.Issued, "already_issued": resp.AlreadyIssued},
			IPAddress:   httpx.ClientIP(r),
			UserAgent:   r.Header.Get("User-Agent"),
		})
		httpx.JSON(w, http.StatusOK, resp)

	case errors.Is(err, election.ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "no such election")

	case errors.Is(err, ErrElectionNotIssuable):
		httpx.Error(w, http.StatusConflict, err.Error())

	default:
		log.Printf("issuing voter credentials failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to issue voter credentials")
	}
}

func (h *Handler) getResults(w http.ResponseWriter, r *http.Request) {
	if _, err := h.authService.Authorize(w, r, dashboardRoles...); err != nil {
		httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	results, err := h.service.GetResults(r.Context(), r.PathValue("id"))
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusOK, results)
	case errors.Is(err, election.ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "no such election")
	case errors.Is(err, ErrResultsNotReady):
		httpx.Error(w, http.StatusConflict, err.Error())
	default:
		log.Printf("getting results failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to get results")
	}
}

func (h *Handler) getTokenStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.service.GetTokenStatus(r.Context(), r.PathValue("token"))
	if err != nil {
		log.Printf("getting token status failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to look up voting link")
		return
	}
	httpx.JSON(w, http.StatusOK, status)
}

func (h *Handler) authenticate(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")

	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	expiresAt, err := h.service.AuthenticateVoter(r.Context(), token, req, httpx.ClientIP(r), r.Header.Get("User-Agent"))
	switch {
	case err == nil:
		status, statusErr := h.service.GetTokenStatus(r.Context(), token)
		if statusErr != nil || status.Election == nil {
			log.Printf("looking up election after authentication failed: %v", statusErr)
			httpx.Error(w, http.StatusInternalServerError, "authenticated, but failed to continue")
			return
		}
		setVoterCookie(w, status.Election.ID, token, expiresAt, h.cookieSecure)
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "authenticated"})

	case errors.Is(err, ErrInvalidOrExpiredToken):
		httpx.Error(w, http.StatusBadRequest, err.Error())

	case errors.Is(err, ErrCredentialLocked):
		httpx.Error(w, http.StatusLocked, err.Error())

	case errors.Is(err, ErrVerificationFailed):
		httpx.Error(w, http.StatusUnauthorized, err.Error())

	case errors.Is(err, ErrAlreadyVoted):
		httpx.Error(w, http.StatusConflict, err.Error())

	default:
		log.Printf("authenticating voter failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to authenticate")
	}
}

func (h *Handler) getBallot(w http.ResponseWriter, r *http.Request) {
	electionID := r.URL.Query().Get("electionId")
	if electionID == "" {
		httpx.Error(w, http.StatusBadRequest, "electionId is required")
		return
	}
	token := readVoterCookie(r, electionID)

	ballot, err := h.service.GetBallot(r.Context(), token)
	switch {
	case err == nil:
		httpx.JSON(w, http.StatusOK, ballot)
	case errors.Is(err, ErrSessionInvalid):
		httpx.Error(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, ErrAlreadyVoted):
		httpx.Error(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrElectionNotLive):
		// Distinct from ErrAlreadyVoted's 409 on purpose - both are states
		// where you can't vote right now, but they're not the same state,
		// and the frontend needs to tell them apart to show the right
		// screen instead of calling everything "already voted".
		httpx.Error(w, http.StatusForbidden, err.Error())
	default:
		log.Printf("getting ballot failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to load ballot")
	}
}

type castVoteRequest struct {
	ElectionID string             `json:"electionId"`
	Selections map[string]*string `json:"selections"`
	// ExplicitNo marks a deliberate "No" on an uncontested portfolio's
	// Yes/No ballot - a "Yes" needs no separate signal, it's just that
	// portfolio's one candidate chosen in Selections as normal.
	ExplicitNo map[string]bool `json:"explicitNo"`
}

func (h *Handler) castVote(w http.ResponseWriter, r *http.Request) {
	var req castVoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ElectionID == "" {
		httpx.Error(w, http.StatusBadRequest, "electionId is required")
		return
	}
	token := readVoterCookie(r, req.ElectionID)

	err := h.service.CastVote(r.Context(), token, req.Selections, req.ExplicitNo)
	switch {
	case err == nil:
		clearVoterCookie(w, req.ElectionID, h.cookieSecure)
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "voted"})
	case errors.Is(err, ErrSessionInvalid):
		httpx.Error(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, ErrAlreadyVoted):
		httpx.Error(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrElectionNotLive):
		httpx.Error(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrInvalidSelection):
		httpx.Error(w, http.StatusBadRequest, err.Error())
	default:
		log.Printf("casting vote failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to cast vote")
	}
}

func voterCookieName(electionID string) string {
	return "voter_token_" + electionID
}

func setVoterCookie(w http.ResponseWriter, electionID, rawToken string, expiresAt time.Time, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     voterCookieName(electionID),
		Value:    rawToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
	})
}

func clearVoterCookie(w http.ResponseWriter, electionID string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     voterCookieName(electionID),
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func readVoterCookie(r *http.Request, electionID string) string {
	cookie, err := r.Cookie(voterCookieName(electionID))
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (h *Handler) logAudit(ctx context.Context, entry audit.Entry) {
	if err := audit.LogStandalone(ctx, h.pool, entry); err != nil {
		log.Printf("failed to write audit log for action %q: %v", entry.Action, err)
	}
}
