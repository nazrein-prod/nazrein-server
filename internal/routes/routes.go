package routes

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	"github.com/grvbrk/track-yt-video/internal/app"
)

func SetupRoutes(app *app.Application) *chi.Mux {
	r := chi.NewRouter()

	r.Use(app.MiddlewareHandler.RequestLogger)
	r.Use(httprate.LimitAll(200, time.Minute))

	r.Route("/auth", func(r chi.Router) {

		r.Use(httprate.LimitAll(100, time.Minute))

		// Auth routes without CORS
		r.Get("/google/login", app.Oauth.Login)
		r.Get("/google/logout", app.Oauth.Logout)
		r.Get("/google/callback", app.Oauth.Callback)

		r.Get("/admin/google/login", app.AdminOauth.Login)
		r.Get("/admin/google/logout", app.AdminOauth.Logout)
		r.Get("/admin/google/callback", app.AdminOauth.Callback)
		r.Get("/admin", app.AdminOauth.AuthAdmin)

		// Auth routes with CORS
		r.Group(func(r chi.Router) {
			r.Use(app.MiddlewareHandler.Cors)
			r.Get("/user", app.Oauth.AuthUser)
		})
	})

	r.Route("/api", func(r chi.Router) {
		r.Use(httprate.LimitAll(50, time.Minute))

		r.Use(app.MiddlewareHandler.Cors)

		// public routes
		r.Get("/videos", app.VideoHandler.HandlerGetVideos)
		r.Get("/videos/{id}", app.VideoHandler.HandlerGetVideoByID)
		r.Get("/videos/autocomplete", app.VideoHandler.HandlerGetSimilarVideosByName)
		r.Get("/videos/analytics/{id}", app.AnalyticsVideoHandler.HandlerGetVideoAnalyticsByID)

		// auth routes
		r.Group(func(r chi.Router) {
			r.Use(app.MiddlewareHandler.Authenticate)

			r.Route("/dashboard", func(r chi.Router) {
				r.Get("/metrics/{user_id}", app.DashboardHandler.HandlerGetDashboardMetrics)
			})

			r.Get("/videos/user/{user_id}", app.VideoHandler.HandlerGetVideosByUserID)
			r.Get("/videos/bookmarks", app.VideoHandler.HandlerGetBookmarkedVideosByUserID)

			r.Route("/request", func(r chi.Router) {
				r.Get("/", app.VideoRequestHandler.HandlerGetAllVideoRequestsByUserID)
				r.Post("/", app.VideoRequestHandler.HandlerCreateVideoRequest)
				r.Delete("/{id}", app.VideoRequestHandler.HandlerDeleteVideoRequestByID)
			})

			r.Route("/bookmark", func(r chi.Router) {
				r.Post("/{id}", app.BookmarkHandler.HandlerCreateBookmark)
				r.Delete("/{id}", app.BookmarkHandler.HandlerDeleteBookmark)
			})
		})
	})

	r.Route("/admin", func(r chi.Router) {
		r.Use(httprate.LimitAll(100, time.Minute))

		r.Use(app.MiddlewareHandler.Cors)
		// r.Use(app.MiddlewareHandler.AuthenticateAdmin)

		r.Route("/request", func(r chi.Router) {
			r.Get("/", app.AdminHandler.HandlerGetVideoRequests)
			r.Post("/", app.AdminHandler.HandlerApproveVideoRequest)
			r.Patch("/{request_id}", app.AdminHandler.HandlerUpdateVideoRequest)
		})
	})

	return r
}
