package app

import (
	"database/sql"
	"log"
	"os"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/grvbrk/track-yt-video/internal/auth"
	"github.com/grvbrk/track-yt-video/internal/handlers"
	handler_analytics "github.com/grvbrk/track-yt-video/internal/handlers/analytics"
	"github.com/grvbrk/track-yt-video/internal/middlewares"
	"github.com/grvbrk/track-yt-video/internal/store"
	"github.com/grvbrk/track-yt-video/internal/store/admin"
	"github.com/grvbrk/track-yt-video/internal/store/analytics"
	"github.com/grvbrk/track-yt-video/migrations"
	"github.com/redis/go-redis/v9"
)

var (
	authKey            = securecookie.GenerateRandomKey(64)
	encryptionKey      = securecookie.GenerateRandomKey(32)
	adminAuthKey       = securecookie.GenerateRandomKey(64)
	adminEncryptionKey = securecookie.GenerateRandomKey(32)
)

type Application struct {
	Logger                *log.Logger
	RedisClient           *redis.Client
	Oauth                 *auth.GoogleOauth
	AdminOauth            *auth.AdminGoogleOauth
	SessionStore          *sessions.CookieStore
	db                    *sql.DB
	DBConn                driver.Conn
	MiddlewareHandler     *middlewares.MiddlwareHandler
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
	sessionStore := sessions.NewCookieStore(authKey, encryptionKey)
	adminSessionStore := sessions.NewCookieStore(adminAuthKey, adminEncryptionKey)

	redisClient, err := store.ConnectRedis()
	if err != nil {
		return nil, err
	}

	pgDB, err := store.ConnectPGDB()
	if err != nil {
		return nil, err
	}

	dbConn, err := store.ConnectClickhouse()
	if err != nil {
		return nil, err
	}

	err = store.MigrateFS(pgDB, migrations.FS, "db")
	if err != nil {
		logger.Println("PANIC: Postgresql migration failed, exiting...")
		panic(err)
	}

	logger.Println("Database migrated...")

	err = store.MigrateClickhouse()
	if err != nil {
		logger.Println("PANIC: Clickhouse migration failed, exiting...")
		panic(err)
	}

	logger.Println("Clickhouse migrated...")

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

	middlewareHandler := middlewares.NewMiddlewareHandler(logger, sessionStore)

	app := &Application{
		Logger:                logger,
		RedisClient:           redisClient,
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
