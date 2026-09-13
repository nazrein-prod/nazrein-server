package routes

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	"github.com/grvbrk/nazrein_server/internal/api"
	"github.com/grvbrk/nazrein_server/internal/app"
	"github.com/grvbrk/nazrein_server/internal/utils"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
)

// specValidator returns middleware that validates every request against the
// embedded OpenAPI spec — path, method, required parameters, and the full JSON
// Schema of each one (minimum/maximum/enum), which plain parameter binding does
// not check.
//
// Authentication is deliberately a no-op here. The spec declares which
// endpoints need a session so that clients and docs are correct, but enforcement
// belongs to the Authenticate / AuthenticateAdmin middleware, which owns the
// session store. Returning an error here instead would reject every request
// before that middleware ever runs.
func specValidator() (func(http.Handler) http.Handler, error) {
	// kin-openapi ignores `format` on strings unless a validator is registered
	// for it. Without this, `format: uuid` is documentation only and a path like
	// /videos/not-a-uuid reaches the binding layer before being rejected.
	openapi3.DefineStringFormat("uuid", openapi3.FormatOfStringForUUIDOfRFC4122)

	spec, err := api.GetSwagger()
	if err != nil {
		return nil, fmt.Errorf("loading embedded openapi spec: %w", err)
	}

	// The spec lists concrete servers; requests arrive with whatever Host the
	// proxy set, so server matching would reject them.
	spec.Servers = nil

	return nethttpmiddleware.OapiRequestValidatorWithOptions(spec, &nethttpmiddleware.Options{
		Options: openapi3filter.Options{
			AuthenticationFunc: func(context.Context, *openapi3filter.AuthenticationInput) error {
				return nil
			},
		},
		ErrorHandlerWithOpts: func(_ context.Context, err error, w http.ResponseWriter, _ *http.Request, opts nethttpmiddleware.ErrorHandlerOpts) {
			utils.WriteJSON(w, opts.StatusCode, utils.Envelope{"message": err.Error()})
		},
	}), nil
}

func SetupRoutes(app *app.Application) (*chi.Mux, error) {
	r := chi.NewRouter()
	adapter := NewServerAdapter(app)

	validate, err := specValidator()
	if err != nil {
		return nil, err
	}

	wrapper := api.ServerInterfaceWrapper{
		Handler: adapter,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": err.Error()})
		},
	}

	r.Use(httprate.LimitAll(200, time.Minute))
	r.Use(app.MiddlewareHandler.RequestLogger)
	r.Use(app.MiddlewareHandler.Security)
	r.Use(validate)

	r.Route("/auth", func(r chi.Router) {

		r.Use(httprate.LimitAll(100, time.Minute))

		// Auth routes without CORS
		r.Get("/google/login", wrapper.UserOAuthLogin)
		r.Get("/google/logout", wrapper.UserOAuthLogout)
		r.Get("/google/callback", wrapper.UserOAuthCallback)

		r.Get("/admin/google/login", wrapper.AdminOAuthLogin)
		r.Get("/admin/google/logout", wrapper.AdminOAuthLogout)
		r.Get("/admin/google/callback", wrapper.AdminOAuthCallback)

		// Auth routes with CORS
		r.Group(func(r chi.Router) {
			r.Use(app.MiddlewareHandler.Cors)
			r.Get("/user", wrapper.AuthUser)
			r.Get("/admin", wrapper.AuthAdmin)
		})
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(httprate.LimitAll(100, time.Minute))
		r.Use(app.MiddlewareHandler.Cors)

		// Public routes. OptionalAuthenticate lets a signed-in visitor get
		// per-user detail (their bookmarks) without making these endpoints
		// require a session.
		r.Route("/public", func(r chi.Router) {
			r.Use(app.MiddlewareHandler.OptionalAuthenticate)

			r.Get("/videos", wrapper.GetPublicVideos)
			r.Get("/videos/{id}", wrapper.GetPublicVideoByID)
			r.Get("/videos/autocomplete", wrapper.GetSimilarVideos)
			r.Get("/videos/analytics/{id}", wrapper.GetVideoAnalyticsByID)
		})

		// auth routes
		r.Group(func(r chi.Router) {
			r.Use(app.MiddlewareHandler.Authenticate)

			r.Route("/dashboard", func(r chi.Router) {
				r.Get("/metrics", wrapper.GetDashboardMetrics)
			})

			r.Get("/videos", wrapper.GetUserVideos)
			r.Get("/videos/bookmarks", wrapper.GetUserBookmarks)

			r.Route("/request", func(r chi.Router) {
				r.Get("/", wrapper.GetUserVideoRequests)
				r.Post("/", wrapper.CreateVideoRequest)
				r.Delete("/{id}", wrapper.DeleteVideoRequest)
			})

			r.Route("/bookmark", func(r chi.Router) {
				r.Post("/{id}", wrapper.CreateBookmark)
				r.Delete("/{id}", wrapper.DeleteBookmark)
			})
		})
	})

	r.Route("/admin", func(r chi.Router) {
		r.Use(httprate.LimitAll(100, time.Minute))
		r.Use(app.MiddlewareHandler.Cors)
		r.Use(app.MiddlewareHandler.AuthenticateAdmin)

		r.Route("/request", func(r chi.Router) {
			r.Get("/", wrapper.GetAdminVideoRequests)
			r.Post("/", wrapper.ApproveVideoRequest)
			r.Patch("/{request_id}", wrapper.UpdateVideoRequest)
		})
	})

	return r, nil
}
