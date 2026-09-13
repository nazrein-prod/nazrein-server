package routes

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/grvbrk/nazrein_server/internal/api"
	"github.com/grvbrk/nazrein_server/internal/app"
	"github.com/grvbrk/nazrein_server/internal/store"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Ensure ServerAdapter implements api.ServerInterface at compile time.
var _ api.ServerInterface = (*ServerAdapter)(nil)

// ServerAdapter binds the oapi-codegen generated ServerInterface to the
// application handlers.
//
// Its job is translation, and only translation: the generated layer hands it
// transport-level types (bound and validated against the OpenAPI spec), and it
// converts them into the domain types the handlers take. Handlers must not
// re-read parameters off the request — if a value is in the spec, it arrives
// here already parsed, and that is the single source of truth.
type ServerAdapter struct {
	app *app.Application
}

// NewServerAdapter creates a new ServerAdapter.
func NewServerAdapter(app *app.Application) *ServerAdapter {
	return &ServerAdapter{app: app}
}

// ---------------------------------------------------------------- public ---

func (s *ServerAdapter) GetPublicVideos(w http.ResponseWriter, r *http.Request, params api.GetPublicVideosParams) {
	// The spec declares defaults, but a default is only applied by a validator
	// that knows the schema — binding alone leaves an absent parameter nil. These
	// fallbacks must stay in step with the defaults in openapi.yaml.
	page := 1
	if params.Page != nil {
		page = *params.Page
	}

	limit := 10
	if params.Limit != nil {
		limit = *params.Limit
	}

	sortBy := store.SortByPopular
	if params.SortBy != nil {
		sortBy = store.ValidateSortBy(string(*params.SortBy))
	}

	searchType := store.SearchVideo
	if params.Type != nil {
		searchType = store.ValidateSearchType(string(*params.Type))
	}

	var query string
	if params.Q != nil {
		query = *params.Q
	}

	s.app.VideoHandler.HandlerGetVideos(w, r, store.GetVideosParams{
		Page:   page,
		Limit:  limit,
		SortBy: sortBy,
		Query:  query,
		Type:   searchType,
	})
}

func (s *ServerAdapter) GetPublicVideoByID(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	s.app.VideoHandler.HandlerGetVideoByID(w, r, uuid.UUID(id))
}

func (s *ServerAdapter) GetSimilarVideos(w http.ResponseWriter, r *http.Request, params api.GetSimilarVideosParams) {
	s.app.VideoHandler.HandlerGetSimilarVideosByName(w, r, params.Q)
}

func (s *ServerAdapter) GetVideoAnalyticsByID(w http.ResponseWriter, r *http.Request, id string, params api.GetVideoAnalyticsByIDParams) {
	limit := 0
	if params.Limit != nil {
		limit = *params.Limit
	}

	offset := 0
	if params.Offset != nil {
		offset = *params.Offset
	}

	s.app.AnalyticsVideoHandler.HandlerGetVideoAnalyticsByID(w, r, id, limit, offset)
}

// ------------------------------------------------------------------ user ---

func (s *ServerAdapter) GetDashboardMetrics(w http.ResponseWriter, r *http.Request) {
	s.app.DashboardHandler.HandlerGetDashboardMetrics(w, r)
}

func (s *ServerAdapter) GetUserVideos(w http.ResponseWriter, r *http.Request) {
	s.app.VideoHandler.HandlerGetVideosByUserID(w, r)
}

func (s *ServerAdapter) GetUserBookmarks(w http.ResponseWriter, r *http.Request) {
	s.app.VideoHandler.HandlerGetBookmarkedVideosByUserID(w, r)
}

func (s *ServerAdapter) GetUserVideoRequests(w http.ResponseWriter, r *http.Request) {
	s.app.VideoRequestHandler.HandlerGetAllVideoRequestsByUserID(w, r)
}

func (s *ServerAdapter) CreateVideoRequest(w http.ResponseWriter, r *http.Request) {
	s.app.VideoRequestHandler.HandlerCreateVideoRequest(w, r)
}

func (s *ServerAdapter) DeleteVideoRequest(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	s.app.VideoRequestHandler.HandlerDeleteVideoRequestByID(w, r, uuid.UUID(id))
}

func (s *ServerAdapter) CreateBookmark(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	s.app.BookmarkHandler.HandlerCreateBookmark(w, r, uuid.UUID(id))
}

func (s *ServerAdapter) DeleteBookmark(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	s.app.BookmarkHandler.HandlerDeleteBookmark(w, r, uuid.UUID(id))
}

// ----------------------------------------------------------------- admin ---

func (s *ServerAdapter) GetAdminVideoRequests(w http.ResponseWriter, r *http.Request) {
	s.app.AdminHandler.HandlerGetVideoRequests(w, r)
}

func (s *ServerAdapter) ApproveVideoRequest(w http.ResponseWriter, r *http.Request) {
	s.app.AdminHandler.HandlerApproveVideoRequest(w, r)
}

func (s *ServerAdapter) UpdateVideoRequest(w http.ResponseWriter, r *http.Request, requestId openapi_types.UUID) {
	s.app.AdminHandler.HandlerUpdateVideoRequest(w, r, uuid.UUID(requestId))
}

// ------------------------------------------------------------------ auth ---

func (s *ServerAdapter) AuthUser(w http.ResponseWriter, r *http.Request) {
	s.app.Oauth.AuthUser(w, r)
}

func (s *ServerAdapter) AuthAdmin(w http.ResponseWriter, r *http.Request) {
	s.app.AdminOauth.AuthAdmin(w, r)
}

// The OAuth endpoints are browser redirects. They read `code` and `state`
// straight off the request because the flow is defined by the provider, not by
// us; params is bound for spec completeness but the handlers own the exchange.
func (s *ServerAdapter) UserOAuthLogin(w http.ResponseWriter, r *http.Request) {
	s.app.Oauth.Login(w, r)
}

func (s *ServerAdapter) UserOAuthLogout(w http.ResponseWriter, r *http.Request) {
	s.app.Oauth.Logout(w, r)
}

func (s *ServerAdapter) UserOAuthCallback(w http.ResponseWriter, r *http.Request, params api.UserOAuthCallbackParams) {
	s.app.Oauth.Callback(w, r)
}

func (s *ServerAdapter) AdminOAuthLogin(w http.ResponseWriter, r *http.Request) {
	s.app.AdminOauth.Login(w, r)
}

func (s *ServerAdapter) AdminOAuthLogout(w http.ResponseWriter, r *http.Request) {
	s.app.AdminOauth.Logout(w, r)
}

func (s *ServerAdapter) AdminOAuthCallback(w http.ResponseWriter, r *http.Request, params api.AdminOAuthCallbackParams) {
	s.app.AdminOauth.Callback(w, r)
}
