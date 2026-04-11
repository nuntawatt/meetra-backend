// Package bootstrap wires all application dependencies together.
// This is the composition root — the ONLY place where concrete types are
// instantiated and injected into interfaces. All other packages depend only
// on interfaces, never on concrete implementations.
package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	internalAuth "github.com/nuntawatt/meetra-backend/internal/auth"
	"github.com/nuntawatt/meetra-backend/internal/config"
	deliveryHTTP "github.com/nuntawatt/meetra-backend/internal/delivery/http"
	"github.com/nuntawatt/meetra-backend/internal/delivery/http/handler"
	ws "github.com/nuntawatt/meetra-backend/internal/delivery/websocket"
	pgRepo "github.com/nuntawatt/meetra-backend/internal/infrastructure/persistence/postgres"
	redisRepo "github.com/nuntawatt/meetra-backend/internal/infrastructure/persistence/redis"
	eventUC "github.com/nuntawatt/meetra-backend/internal/usecase/event"
	notifUC "github.com/nuntawatt/meetra-backend/internal/usecase/notification"
	userUC "github.com/nuntawatt/meetra-backend/internal/usecase/user"
	"github.com/nuntawatt/meetra-backend/internal/worker"
	"github.com/nuntawatt/meetra-backend/pkg/logger"
)

// App is the fully initialised application container.
type App struct {
	Router     *deliveryHTTP.RouterConfig
	DB         *sqlx.DB
	Redis      *redis.Client
	WorkerPool *worker.Pool
	Logger     *zap.Logger
}

// New builds the application, wires all dependencies, and runs migrations.
// This is the single entry point for dependency injection.
func New() (*App, error) {
	// ——— Config ——————————————————————————————————————————————————————————————
	appCfg := config.LoadApp()
	dbCfg := config.LoadDB()
	redisCfg := config.LoadRedis()
	authCfg := config.LoadAuth()

	// ——— Logger ——————————————————————————————————————————————————————————————
	logger.Init(appCfg.Env)
	log := logger.Get()

	// ——— PostgreSQL ——————————————————————————————————————————————————————————
	db, err := connectPostgres(dbCfg)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: postgres: %w", err)
	}
	log.Info("postgres connected", zap.String("host", dbCfg.Host))

	// Automatically apply pending migrations on startup
	if err := runMigrations(dbCfg.DSN()); err != nil {
		// ErrNoChange is fine — it just means DB is already up to date
		log.Warn("migrate", zap.Error(err))
	} else {
		log.Info("migrations applied")
	}

	// ——— Redis ———————————————————————————————————————————————————————————————
	redisClient, err := connectRedis(redisCfg)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: redis: %w", err)
	}
	log.Info("redis connected", zap.String("addr", redisCfg.Addr()))

	// ——— Auth services ———————————————————————————————————————————————————————
	jwtSvc := internalAuth.NewJWTService(authCfg.AccessSecret, authCfg.RefreshSecret)
	bcryptSvc := internalAuth.NewBcryptService(12) // cost 12 is safe for production

	// ——— Repositories ————————————————————————————————————————————————————————
	// Each concrete repo satisfies the interface defined in its use-case package.
	userRepo := pgRepo.NewUserRepo(db)
	eventRepo := pgRepo.NewEventRepo(db)
	notifRepo := pgRepo.NewNotificationRepo(db)
	cacheRepo := redisRepo.NewCacheRepo(redisClient)

	// ——— WebSocket Hub ———————————————————————————————————————————————————————
	// The hub must be created before the notification use-case because it IS the Publisher.
	hub := ws.NewHub(log)

	// ——— Use-Cases ———————————————————————————————————————————————————————————
	// Wire inner layers first (notification has no deps on other use-cases)
	notifUseCase := notifUC.New(notifRepo, hub)

	userUseCase := userUC.New(
		userRepo,
		cacheRepo.AsUserCache(),
		authCfg,
		jwtSvc,
		bcryptSvc,
	)

	eventUseCase := eventUC.New(
		eventRepo,
		cacheRepo.AsEventCache(),
		notifUseCase,
	)

	// ——— HTTP Handlers ———————————————————————————————————————————————————————
	userHandler := handler.NewUserHandler(userUseCase)
	eventHandler := handler.NewEventHandler(eventUseCase)
	uploadHandler := handler.NewUploadHandler(appCfg.UploadPath)

	// ——— Worker Pool —————————————————————————————————————————————————————————
	// 5 workers, queue depth of 50 pending jobs
	pool := worker.New(5, 50, log)
	log.Info("worker pool started", zap.Int("workers", 5))

	routerCfg := &deliveryHTTP.RouterConfig{
		UserHandler:   userHandler,
		EventHandler:  eventHandler,
		UploadHandler: uploadHandler,
		WSHub:         hub,
		JWTService:    jwtSvc,
		RedisClient:   redisClient,
		Logger:        log,
	}

	return &App{
		Router:     routerCfg,
		DB:         db,
		Redis:      redisClient,
		WorkerPool: pool,
		Logger:     log,
	}, nil
}

// ——— Infrastructure helpers ——————————————————————————————————————————————————

func connectPostgres(cfg config.DBConfig) (*sqlx.DB, error) {
	db, err := sqlx.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, err
	}

	// Connection pool tuning
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return db, nil
}

func connectRedis(cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return client, nil
}

func runMigrations(dsn string) error {
	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}
