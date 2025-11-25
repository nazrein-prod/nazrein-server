package app

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/grvbrk/nazrein_server/internal/auth"
	"github.com/grvbrk/nazrein_server/internal/handlers"
	handler_analytics "github.com/grvbrk/nazrein_server/internal/handlers/analytics"
	"github.com/grvbrk/nazrein_server/internal/middlewares"
	"github.com/grvbrk/nazrein_server/internal/services"
	"github.com/grvbrk/nazrein_server/internal/store"
	"github.com/grvbrk/nazrein_server/internal/store/admin"
	"github.com/grvbrk/nazrein_server/internal/store/analytics"
	// "github.com/grvbrk/nazrein_server/migrations"
)

var (
	authKey            = securecookie.GenerateRandomKey(64)
	encryptionKey      = securecookie.GenerateRandomKey(32)
	adminAuthKey       = securecookie.GenerateRandomKey(64)
	adminEncryptionKey = securecookie.GenerateRandomKey(32)
)

type Application struct {
	Logger *log.Logger
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
	logger := log.New(os.Stdout, "LOGGING: ", log.Ldate|log.Ltime)
	adminLogger := log.New(os.Stdout, "ADMIN LOGGING: ", log.Ldate|log.Ltime)

	pgDB, err := services.ConnectPGDB()
	if err != nil {
		logger.Println("Error connecting to db")
		return nil, err
	}

	dbConn, err := services.ConnectClickhouse()
	if err != nil {
		logger.Println("Error connecting to clickhouse")
		return nil, err
	}

	// err = store.MigrateFS(pgDB, migrations.FS, "db")
	// if err != nil {
	// 	logger.Println("PANIC: Postgresql migration failed, exiting...")
	// 	panic(err)
	// }

	// logger.Println("Database migrated...")

	err = services.MigrateClickhouse()
	if err != nil {
		logger.Println("PANIC: Clickhouse migration failed, exiting...")
		return nil, err
	}

	env := os.Getenv("ENV")
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

	if env == "production" {
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

	sessionStore := sessions.NewCookieStore(authKey, encryptionKey)
	sessionStore.Options = userOptions

	adminSessionStore := sessions.NewCookieStore(adminAuthKey, adminEncryptionKey)
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
