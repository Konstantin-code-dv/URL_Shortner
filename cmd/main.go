package main

import (
	"URL_Shortner/internal/config"
	"URL_Shortner/internal/http-server/handlers/redirect"
	"URL_Shortner/internal/http-server/handlers/url/save"
	mwLogger "URL_Shortner/internal/http-server/middleware/logger"
	"URL_Shortner/internal/lib/logger/handlers/slogpretty"
	"URL_Shortner/internal/lib/logger/sl"
	"URL_Shortner/internal/storage/sqlite"
	"log"
	"log/slog"
	"net/http"
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
	//router.Use(middleware.Logger)
	router.Use(mwLogger.New(log)) // кастомный логгер
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)
	router.Use(middleware.Timeout(60 * time.Second))

	router.Route("/url", func(r chi.Router) {
		r.Use(middleware.BasicAuth("url-shortener", map[string]string{
			cfg.HTTPServer.User: cfg.HTTPServer.Password,
		}))

		r.Post("/", save.New(log, storage))
		// TODO: add DELETE /url/{id}
	})

	router.Post("/url", save.New(log, storage))
	router.Get("/{alias}", redirect.New(log, storage))

	//Запуск сервера
	log.Info("starting http server", slog.String("addres:", cfg.Address))
	server := &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}
	if err := server.ListenAndServe(); err != nil {
		slog.Error("failed to start http server", sl.Err(err))
		os.Exit(1)
	}

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
