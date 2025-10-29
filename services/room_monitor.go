package services

import (
	"context"
	"time"

	"poker-engine/config"
	"poker-engine/storage"
	"poker-engine/utils"
)

// RoomMonitor - сервис для мониторинга всех комнат
type RoomMonitor struct {
	redis            *storage.RedisClient
	config           *config.EngineConfig
	logger           *utils.Logger
	ctx              context.Context
	cancelFunc       context.CancelFunc
	gameStateService *GameStateService
	actionLogger     *ActionLogger
	deckManager      *DeckManager
	cardDealer       *CardDealer
	ticker           *time.Ticker
	isRunning        bool
}

// NewRoomMonitor - создает новый экземпляр RoomMonitor
func NewRoomMonitor(
	redis *storage.RedisClient,
	cfg *config.EngineConfig,
	gameStateService *GameStateService,
	actionLogger *ActionLogger,
) *RoomMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	return &RoomMonitor{
		redis:            redis,
		config:           cfg,
		logger:           utils.NewLogger("RoomMonitor"),
		ctx:              ctx,
		cancelFunc:       cancel,
		gameStateService: gameStateService,
		actionLogger:     actionLogger,
		isRunning:        false,
	}
}

// Start - запускает мониторинг комнат
func (rm *RoomMonitor) Start() {
	if rm.isRunning {
		rm.logger.Warning("Мониторинг уже запущен")
		return
	}

	rm.isRunning = true
	rm.logger.Success("Мониторинг комнат запущен")
	rm.logger.Infof("Интервал проверки: %v", rm.config.CheckInterval)
	rm.logger.Infof("Минимум игроков для старта: %d", rm.config.MinPlayersToStart)

	rm.ticker = time.NewTicker(rm.config.CheckInterval)
	defer rm.ticker.Stop()

	rm.checkAllRooms()

	for {
		select {
		case <-rm.ticker.C:
			rm.checkAllRooms()
		case <-rm.ctx.Done():
			rm.logger.Info("Контекст отменен, останавливаем мониторинг")
			rm.isRunning = false
			return
		}
	}
}

// Stop - останавливает мониторинг комнат
func (rm *RoomMonitor) Stop() {
	if !rm.isRunning {
		rm.logger.Warning("Мониторинг не запущен")
		return
	}

	rm.logger.Info("Останавливаем мониторинг комнат...")
	rm.cancelFunc()

	time.Sleep(100 * time.Millisecond)

	if rm.ticker != nil {
		rm.ticker.Stop()
	}

	rm.isRunning = false
	rm.logger.Success("Мониторинг комнат остановлен")
}

// IsRunning - возвращает статус работы мониторинга
func (rm *RoomMonitor) IsRunning() bool {
	return rm.isRunning
}

// checkAllRooms - проверяет все активные комнаты во всех клубах
func (rm *RoomMonitor) checkAllRooms() {
	pattern := rm.redis.GetKeys().ClubRoomsActivePattern()
	iter := rm.redis.Scan(pattern)
	roomsChecked := 0

	for iter.Next(rm.ctx) {
		clubRoomsKey := iter.Val()
		clubID := rm.redis.GetKeys().ExtractClubID(clubRoomsKey)
		if clubID == "" {
			rm.logger.Warningf("Не удалось извлечь clubId из ключа: %s", clubRoomsKey)
			continue
		}

		roomIDs, err := rm.redis.ZRange(clubRoomsKey, 0, -1)
		if err != nil {
			rm.logger.Errorf("Ошибка при получении комнат клуба %s: %v", clubID, err)
			continue
		}

		for _, roomID := range roomIDs {
			rm.checkRoom(clubID, roomID)
			roomsChecked++
		}
	}

	if err := iter.Err(); err != nil {
		rm.logger.Errorf("Ошибка при сканировании клубов: %v", err)
	}

	if roomsChecked > 0 {
		rm.logger.Debugf("Проверено комнат: %d", roomsChecked)
	}
}

// checkRoom - проверяет состояние конкретной комнаты
func (rm *RoomMonitor) checkRoom(clubID, roomID string) {
	playersCount, err := rm.gameStateService.GetPlayersCount(clubID, roomID)
	if err != nil {
		rm.logger.Errorf("Ошибка при получении количества игроков в комнате %s:%s: %v", clubID, roomID, err)
		return
	}

	game, err := rm.gameStateService.GetGameState(clubID, roomID)
	if err != nil {
		rm.logger.Errorf("Ошибка при получении состояния игры %s:%s: %v", clubID, roomID, err)
		return
	}

	if game == nil {
		return
	}

	currentPhase := game.Phase

	// СЛУЧАЙ 1: Достаточно игроков (≥2) и игра не началась -> ЗАПУСКАЕМ
	if playersCount >= int64(rm.config.MinPlayersToStart) && currentPhase == "waiting" {
		rm.logger.Debugf("Комната %s:%s готова к запуску (игроков: %d)", clubID, roomID, playersCount)
		err := rm.handleGameStart(clubID, roomID, int(playersCount))
		if err != nil {
			rm.logger.Errorf("Ошибка при запуске игры %s:%s: %v", clubID, roomID, err)
		}
		return
	}

	// СЛУЧАЙ 2: Недостаточно игроков (<2) и игра идет -> ОСТАНАВЛИВАЕМ
	if playersCount < int64(rm.config.MinPlayersToStart) && currentPhase != "waiting" {
		rm.logger.Debugf("Комната %s:%s: недостаточно игроков (осталось: %d)", clubID, roomID, playersCount)
		err := rm.handleGameStop(clubID, roomID, string(currentPhase), "insufficient_players")
		if err != nil {
			rm.logger.Errorf("Ошибка при остановке игры %s:%s: %v", clubID, roomID, err)
		}
		return
	}

	// СЛУЧАЙ 3: Игра идет нормально
	if playersCount >= int64(rm.config.MinPlayersToStart) && currentPhase != "waiting" {
		// rm.logger.GameInProgress(clubID, roomID, string(currentPhase), int(playersCount))
	}
}

// handleGameStart - внутренняя логика запуска игры
func (rm *RoomMonitor) handleGameStart(clubID, roomID string, playersCount int) error {
	rm.logger.Infof("Запуск игры в комнате %s:%s", clubID, roomID)

	gameID := utils.GetCurrentTime().Format("20060102150405") + "_" + clubID + "_" + roomID
	startedAt := utils.GetISO8601Time()

	pipe := rm.redis.Pipeline()
	roomInfoKey := rm.redis.GetKeys().RoomInfo(clubID, roomID)
	gameStateKey := rm.redis.GetKeys().GameState(clubID, roomID)
	ctx := rm.redis.GetContext()

	pipe.HSet(ctx, roomInfoKey, "status", "gaming")
	pipe.HSet(ctx, gameStateKey, "phase", "pre_flop")
	pipe.HSet(ctx, gameStateKey, "game_id", gameID)
	pipe.HSet(ctx, gameStateKey, "started_at", startedAt)
	pipe.HSet(ctx, gameStateKey, "pot", 0)
	pipe.HSet(ctx, gameStateKey, "current_bet", 0)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}

	rm.actionLogger.LogGameStarted(clubID, roomID, gameID, playersCount)
	rm.logger.GameStarted(clubID, roomID, gameID, playersCount)

	return nil
}

// handleGameStop - внутренняя логика остановки игры
func (rm *RoomMonitor) handleGameStop(clubID, roomID, previousPhase, reason string) error {
	rm.logger.Infof("Остановка игры в комнате %s:%s", clubID, roomID)

	pipe := rm.redis.Pipeline()
	roomInfoKey := rm.redis.GetKeys().RoomInfo(clubID, roomID)
	gameStateKey := rm.redis.GetKeys().GameState(clubID, roomID)
	ctx := rm.redis.GetContext()

	pipe.HSet(ctx, roomInfoKey, "status", "waiting")
	pipe.HSet(ctx, gameStateKey, "phase", "waiting")
	pipe.HSet(ctx, gameStateKey, "game_id", "")
	pipe.HSet(ctx, gameStateKey, "started_at", "")
	pipe.HSet(ctx, gameStateKey, "pot", 0)
	pipe.HSet(ctx, gameStateKey, "current_bet", 0)
	pipe.HSet(ctx, gameStateKey, "community_cards", "[]")

	_, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}

	rm.actionLogger.LogGameStopped(clubID, roomID, previousPhase, reason)
	rm.logger.GameStopped(clubID, roomID, previousPhase, reason)

	return nil
}

// GetStatistics - возвращает статистику мониторинга
func (rm *RoomMonitor) GetStatistics() map[string]interface{} {
	return map[string]interface{}{
		"is_running":      rm.isRunning,
		"check_interval":  rm.config.CheckInterval.String(),
		"min_players":     rm.config.MinPlayersToStart,
		"redis_connected": rm.redis.IsConnected(),
	}
}

// HealthCheck - проверяет здоровье мониторинга
func (rm *RoomMonitor) HealthCheck() error {
	if !rm.isRunning {
		return ErrMonitorNotRunning
	}

	if err := rm.redis.HealthCheck(); err != nil {
		return err
	}

	return nil
}

// ForceCheck - принудительно запускает проверку всех комнат
func (rm *RoomMonitor) ForceCheck() {
	rm.logger.Info("Принудительная проверка всех комнат...")
	rm.checkAllRooms()
	rm.logger.Success("Принудительная проверка завершена")
}

// CheckSpecificRoom - проверяет конкретную комнату
func (rm *RoomMonitor) CheckSpecificRoom(clubID, roomID string) error {
	rm.logger.Infof("Проверка комнаты %s:%s...", clubID, roomID)

	exists, err := rm.gameStateService.RoomExists(clubID, roomID)
	if err != nil {
		return err
	}

	if !exists {
		rm.logger.Warningf("Комната %s:%s не существует", clubID, roomID)
		return ErrRoomNotFound
	}

	rm.checkRoom(clubID, roomID)
	rm.logger.Successf("Проверка комнаты %s:%s завершена", clubID, roomID)
	return nil
}

var (
	ErrMonitorNotRunning = &MonitorError{message: "monitor is not running"}
	ErrRoomNotFound      = &MonitorError{message: "room not found"}
)

type MonitorError struct {
	message string
}

func (e *MonitorError) Error() string {
	return "monitor error: " + e.message
}
