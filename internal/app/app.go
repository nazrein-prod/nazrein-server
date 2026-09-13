package app

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/gorilla/sessions"
	"github.com/grvbrk/nazrein_server/internal/auth"
	"github.com/grvbrk/nazrein_server/internal/handlers"
	handler_analytics "github.com/grvbrk/nazrein_server/internal/handlers/analytics"
	applogger "github.com/grvbrk/nazrein_server/internal/logger"
	"github.com/grvbrk/nazrein_server/internal/middlewares"
	"github.com/grvbrk/nazrein_server/internal/services"
	"github.com/grvbrk/nazrein_server/internal/store"
	"github.com/grvbrk/nazrein_server/internal/store/admin"
	"github.com/grvbrk/nazrein_server/internal/store/analytics"
	"github.com/grvbrk/nazrein_server/internal/utils"
	"github.com/grvbrk/nazrein_server/migrations"
)

type Application struct {
	Logger *slog.Logger
	// RedisClient           *redis.Client
	Oauth                 *auth.GoogleOauth
	AdminOauth            *auth.AdminGoogleOauth
	SessionStore          *sessions.CookieStore
	db                    *sql.DB
	DBConn                driver.Conn
	MiddlewareHandler     *middlewares.MiddlewareHandler
	UserHandler           *handlers.UserHandler
	DashboardHandler      *handlers.DashboardHandler
	VideoHandler          *handlers.VideoHandler
	VideoRequestHandler   *handlers.VideoRequestHandler
	BookmarkHandler       *handlers.BookmarkHandler
	AnalyticsVideoHandler *handler_analytics.AnalyticsVideoHandler
	AdminHandler          *handlers.AdminHandler
}

func NewApplication() (*Application, error) {
	logger := applogger.New("server")
	adminLogger := applogger.New("admin")

	pgDB, err := services.ConnectPGDB()
	if err != nil {
		logger.Error("Error connecting to db", "err", err)
		return nil, err
	}

	dbConn, err := services.ConnectClickhouse()
	if err != nil {
		logger.Error("Error connecting to clickhouse", "err", err)
		return nil, err
	}

	err = services.MigrateFS(pgDB, migrations.FS, "db")
	if err != nil {
		logger.Error("Postgres migration failed, exiting...", "err", err)
		return nil, fmt.Errorf("postgres migration: %w", err)
	}

	logger.Info("Database migrated...")

	err = services.MigrateClickhouse()
	if err != nil {
		logger.Error("Clickhouse migration failed, exiting...", "err", err)
		return nil, err
	}

	env := os.Getenv("ENV")
	production := env == "production"
	var userOptions = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
	}

	var adminOptions = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
	}

	if production {
		userOptions.Secure = true
		userOptions.SameSite = http.SameSiteNoneMode
		userOptions.Domain = ".nazrein.dev"

		adminOptions.Secure = true
		adminOptions.SameSite = http.SameSiteNoneMode
		adminOptions.Domain = ".nazrein.dev"
	} else {
		userOptions.Secure = false
		userOptions.SameSite = http.SameSiteLaxMode
		userOptions.Domain = ""

		adminOptions.Secure = false
		adminOptions.SameSite = http.SameSiteLaxMode
		adminOptions.Domain = ""
	}

	userAuthKey, userEncKey, err := utils.SessionKeys(logger, "SESSION_AUTH_KEY", "SESSION_ENCRYPTION_KEY", production)
	if err != nil {
		return nil, err
	}

	adminAuthKey, adminEncKey, err := utils.SessionKeys(adminLogger, "ADMIN_SESSION_AUTH_KEY", "ADMIN_SESSION_ENCRYPTION_KEY", production)
	if err != nil {
		return nil, err
	}

	sessionStore := sessions.NewCookieStore(userAuthKey, userEncKey)
	sessionStore.Options = userOptions

	adminSessionStore := sessions.NewCookieStore(adminAuthKey, adminEncKey)
	adminSessionStore.Options = adminOptions

	userStore := store.NewPostgresUserStore(pgDB)
	dashboardStore := store.NewPostgresDashboardStore(pgDB)
	videoStore := store.NewPostgresVideoStore(pgDB)
	// redisVideoStore := store.NewRedisVideoStore(redisClient)
	videoRequestStore := store.NewPostgresVideoRequestStore(pgDB)
	bookmarkStore := store.NewPostgresBookmarkStore(pgDB)

	analyticsVideoStore := analytics.NewClickhouseVideoStore(dbConn)

	adminVideoStore := admin.NewPostgresAdminVideoStore(pgDB)
	adminUserStore := admin.NewPostgresAdminUserStore(pgDB)
	adminVideoRequestStore := admin.NewPostgresAdminVideoRequestStore(pgDB)

	oauth, err := auth.NewGoogleOauth(logger, sessionStore, userStore)
	if err != nil {
		return nil, err
	}

	adminoauth, err := auth.NewAdminGoogleOauth(adminLogger, adminSessionStore, userStore)
	if err != nil {
		return nil, err
	}

	userHandler := handlers.NewUserHandler(userStore, logger)
	dashboardHandler := handlers.NewDashboardHandler(dashboardStore, logger)
	videoHandler := handlers.NewVideoHandler(videoStore, logger, oauth)
	videoRequestHandler := handlers.NewVideoRequestHandler(videoRequestStore, logger, oauth)
	bookmarkHandler := handlers.NewBookmarkHandler(videoStore, bookmarkStore, userStore, oauth, logger)

	analyticsVideoHandler := handler_analytics.NewAnalyticsVideoHandler(analyticsVideoStore, logger)

	adminHander := handlers.NewAdminHandler(adminVideoStore, adminUserStore, adminVideoRequestStore, adminLogger, adminoauth)

	middlewareHandler := middlewares.NewMiddlewareHandler(logger, adminLogger, sessionStore, adminSessionStore)

	app := &Application{
		Logger: logger,
		// RedisClient:           redisClient,
		Oauth:                 oauth,
		AdminOauth:            adminoauth,
		SessionStore:          sessionStore,
		db:                    pgDB,
		DBConn:                dbConn,
		MiddlewareHandler:     middlewareHandler,
		UserHandler:           userHandler,
		DashboardHandler:      dashboardHandler,
		VideoHandler:          videoHandler,
		VideoRequestHandler:   videoRequestHandler,
		BookmarkHandler:       bookmarkHandler,
		AnalyticsVideoHandler: analyticsVideoHandler,
		AdminHandler:          adminHander,
	}

	return app, nil

}
