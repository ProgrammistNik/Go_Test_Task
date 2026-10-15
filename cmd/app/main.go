package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ProgrammistNik/Go_Test_Task/internal/agent"
	"github.com/ProgrammistNik/Go_Test_Task/internal/config"
	"github.com/ProgrammistNik/Go_Test_Task/internal/httpapi"
	"github.com/ProgrammistNik/Go_Test_Task/internal/service"
	"github.com/ProgrammistNik/Go_Test_Task/internal/store"
	"github.com/ProgrammistNik/Go_Test_Task/internal/telegram"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	database, err := store.NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer database.Close()
	expenseService := service.NewExpenseService(database)
	handler := httpapi.NewHandler(expenseService, database)
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler.Router(cfg.WebDir),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	if cfg.TelegramToken != "" {
		var llm agent.LLM
		if cfg.LLMAPIKey != "" {
			llm = agent.NewOpenAIClient(cfg.LLMBaseURL, cfg.LLMAPIKey, cfg.LLMModel)
		}
		expenseAgent := agent.New(agent.NewInterpreter(llm), expenseService)
		bot := telegram.NewBot(telegram.NewClient(cfg.TelegramToken), expenseAgent, logger)
		go bot.Run(ctx)
		logger.Info("telegram bot started")
	}
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("http server started", "address", cfg.HTTPAddr)
		serverErrors <- server.ListenAndServe()
	}()
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}
