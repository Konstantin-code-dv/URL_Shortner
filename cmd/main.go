package main

import (
	"URL_Shortner/internal/config"
	mwLogger "URL_Shortner/internal/http-server/middleware/logger"
	"URL_Shortner/internal/lib/logger/handlers/slogpretty"
	"URL_Shortner/internal/lib/logger/sl"
	"URL_Shortner/internal/storage/sqlite"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found")
	}
	//Чтения конфига
	cfg := config.MustLoadConfig()

	//Инициализация логгера
	log := setupLogger(cfg.Env)

	log.Info("starting url-server", slog.String("env", cfg.Env))
	log.Debug("debug logging enabled")

	//fmt.Printf("%#v\n", cfg)

	//Инициализация БД
	storage, err := sqlite.NewStorage(cfg.StoragePath)
	if err != nil {
		slog.Error("failed to init storage", sl.Err(err))
		os.Exit(1)
	}

	log.Info("database created and initialized")
	_ = storage

	//Инициилизация роутера
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(mwLogger.New(log)) // кастомный логгер
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)
	router.Use(middleware.Timeout(60 * time.Second))

}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = setupPrettySlog()
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	default: // If env config is invalid, set prod settings by default due to security
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}

func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}
