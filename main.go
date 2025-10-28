package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"poker-engine/config"
	"poker-engine/services"
	"poker-engine/storage"
	"poker-engine/utils"
)

// Application - главная структура приложения
type Application struct {
	// config - конфигурация приложения
	config *config.Config

	// redis - клиент Redis
	redis *storage.RedisClient

	// services - бизнес-логика
	gameStateService *services.GameStateService
	actionLogger     *services.ActionLogger
	roomMonitor      *services.RoomMonitor

	// logger - главный логгер
	logger *utils.Logger

	// ctx - контекст приложения
	ctx context.Context

	// cancelFunc - функция отмены контекста
	cancelFunc context.CancelFunc

	// shutdownChan - канал для graceful shutdown
	shutdownChan chan os.Signal
}

// NewApplication - создает новый экземпляр приложения
func NewApplication() (*Application, error) {
	// Создаем главный логгер
	logger := utils.NewLogger("Application")

	// Выводим красивый баннер
	logger.PrintHeader("POKER GAME ENGINE")

	// === ШАГ 1: ЗАГРУЗКА КОНФИГУРАЦИИ ===
	logger.Info("Загрузка конфигурации...")
	cfg := config.Load()

	// Валидация конфигурации
	if err := cfg.Validate(); err != nil {
		logger.Errorf("Ошибка валидации конфигурации: %v", err)
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	logger.Success("Конфигурация загружена и валидна")
	logger.Infof("  Redis: %s (DB: %d)", cfg.Redis.Addr, cfg.Redis.DB)
	logger.Infof("  Интервал проверки: %v", cfg.Engine.CheckInterval)
	logger.Infof("  Минимум игроков: %d", cfg.Engine.MinPlayersToStart)

	// === ШАГ 2: ПОДКЛЮЧЕНИЕ К REDIS ===
	logger.PrintSeparator()
	logger.Info("Подключение к Redis...")

	ctx, cancel := context.WithCancel(context.Background())

	redis, err := storage.NewRedisClient(ctx, &cfg.Redis)
	if err != nil {
		logger.Errorf("Не удалось подключиться к Redis: %v", err)
		cancel()
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	// === ШАГ 3: ИНИЦИАЛИЗАЦИЯ СЕРВИСОВ ===
	logger.PrintSeparator()
	logger.Info("Инициализация сервисов...")

	// Создаем сервис состояния игры
	gameStateService := services.NewGameStateService(redis)
	logger.Success("  ✓ GameStateService")

	// Создаем логгер действий
	actionLogger := services.NewActionLogger(redis)
	logger.Success("  ✓ ActionLogger")

	// Создаем мониторинг комнат
	roomMonitor := services.NewRoomMonitor(redis, &cfg.Engine, gameStateService, actionLogger)
	logger.Success("  ✓ RoomMonitor")

	logger.Success("Все сервисы инициализированы")

	// === ШАГ 4: НАСТРОЙКА GRACEFUL SHUTDOWN ===
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	return &Application{
		config:           cfg,
		redis:            redis,
		gameStateService: gameStateService,
		actionLogger:     actionLogger,
		roomMonitor:      roomMonitor,
		logger:           logger,
		ctx:              ctx,
		cancelFunc:       cancel,
		shutdownChan:     shutdownChan,
	}, nil
}

// Start - запускает приложение
func (app *Application) Start() error {
	app.logger.PrintSeparator()
	app.logger.EngineStarted()
	app.logger.PrintSeparator()

	// Запускаем мониторинг комнат в отдельной горутине
	go app.roomMonitor.Start()

	// Ждем сигнала завершения
	<-app.shutdownChan

	// Начинаем graceful shutdown
	return app.Shutdown()
}

// Shutdown - корректно завершает работу приложения
func (app *Application) Shutdown() error {
	app.logger.PrintSeparator()
	app.logger.EngineShuttingDown()

	// Таймаут для shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(),
		app.config.Engine.ShutdownTimeout,
	)
	defer shutdownCancel()

	// Канал для отслеживания завершения
	done := make(chan bool, 1)

	go func() {
		// === ШАГ 1: ОСТАНОВКА МОНИТОРИНГА ===
		app.logger.Info("Останавливаем мониторинг комнат...")
		app.roomMonitor.Stop()
		app.logger.Success("  ✓ Мониторинг остановлен")

		// === ШАГ 2: ЗАКРЫТИЕ REDIS ===
		app.logger.Info("Закрываем соединение с Redis...")
		if err := app.redis.Close(); err != nil {
			app.logger.Errorf("Ошибка при закрытии Redis: %v", err)
		} else {
			app.logger.Success("  ✓ Redis соединение закрыто")
		}

		// === ШАГ 3: ОТМЕНА КОНТЕКСТА ===
		app.cancelFunc()

		done <- true
	}()

	// Ждем завершения или таймаута
	select {
	case <-done:
		app.logger.PrintSeparator()
		app.logger.EngineStopped()
		app.logger.Success("Graceful shutdown завершен успешно")
		return nil

	case <-shutdownCtx.Done():
		app.logger.PrintSeparator()
		app.logger.Warning("Таймаут shutdown превышен, принудительное завершение")
		app.cancelFunc()
		return fmt.Errorf("shutdown timeout exceeded")
	}
}

// HealthCheck - проверяет здоровье приложения
func (app *Application) HealthCheck() error {
	// Проверяем Redis
	if err := app.redis.HealthCheck(); err != nil {
		return fmt.Errorf("redis health check failed: %w", err)
	}

	// Проверяем мониторинг
	if err := app.roomMonitor.HealthCheck(); err != nil {
		return fmt.Errorf("monitor health check failed: %w", err)
	}

	return nil
}

// GetStats - возвращает статистику приложения
func (app *Application) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"redis_connected": app.redis.IsConnected(),
		"monitor_running": app.roomMonitor.IsRunning(),
		"monitor_stats":   app.roomMonitor.GetStatistics(),
		"uptime":          time.Since(time.Now()).String(),
	}
}

// === MAIN ФУНКЦИЯ ===

func main() {
	// Создаем приложение
	app, err := NewApplication()
	if err != nil {
		fmt.Printf("❌ Ошибка инициализации приложения: %v\n", err)
		os.Exit(1)
	}

	// Запускаем приложение
	if err := app.Start(); err != nil {
		fmt.Printf("❌ Ошибка при работе приложения: %v\n", err)
		os.Exit(1)
	}

	// Приложение завершено успешно
	os.Exit(0)
}
