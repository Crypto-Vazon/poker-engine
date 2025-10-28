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
	// redis - клиент для работы с Redis
	redis *storage.RedisClient

	// config - конфигурация движка
	config *config.EngineConfig

	// logger - логгер для вывода сообщений
	logger *utils.Logger

	// ctx - контекст для управления жизненным циклом
	ctx context.Context

	// cancelFunc - функция для отмены контекста
	cancelFunc context.CancelFunc

	// gameStateService - сервис для работы с состоянием игры
	gameStateService *GameStateService

	// actionLogger - сервис для записи действий
	actionLogger *ActionLogger

	// ticker - таймер для периодической проверки
	ticker *time.Ticker

	// isRunning - флаг работы мониторинга
	isRunning bool

	// Обработчики вынесены в отдельные функции (без импорта handlers)
}

// NewRoomMonitor - создает новый экземпляр RoomMonitor
func NewRoomMonitor(
	redis *storage.RedisClient,
	cfg *config.EngineConfig,
	gameStateService *GameStateService,
	actionLogger *ActionLogger,
) *RoomMonitor {
	// Создаем контекст с возможностью отмены
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

	// Создаем ticker для периодической проверки
	rm.ticker = time.NewTicker(rm.config.CheckInterval)
	defer rm.ticker.Stop()

	// Выполняем первую проверку сразу (не ждем первого tick)
	rm.checkAllRooms()

	// Основной цикл мониторинга
	for {
		select {
		// Когда ticker срабатывает - проверяем комнаты
		case <-rm.ticker.C:
			rm.checkAllRooms()

		// Когда контекст отменен - завершаем работу
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

	// Ждем немного, чтобы текущая итерация завершилась
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
	// Получаем паттерн для поиска всех клубов с активными комнатами
	pattern := rm.redis.GetKeys().ClubRoomsActivePattern()

	// Сканируем Redis по паттерну
	iter := rm.redis.Scan(pattern)

	// Счетчик проверенных комнат для статистики
	roomsChecked := 0

	// Проходим по всем найденным ключам
	for iter.Next(rm.ctx) {
		clubRoomsKey := iter.Val() // Например: "club:1:rooms:active"

		// Извлекаем clubId из ключа
		clubID := rm.redis.GetKeys().ExtractClubID(clubRoomsKey)
		if clubID == "" {
			rm.logger.Warningf("Не удалось извлечь clubId из ключа: %s", clubRoomsKey)
			continue
		}

		// Получаем все активные комнаты этого клуба
		roomIDs, err := rm.redis.ZRange(clubRoomsKey, 0, -1)
		if err != nil {
			rm.logger.Errorf("Ошибка при получении комнат клуба %s: %v", clubID, err)
			continue
		}

		// Проверяем каждую комнату
		for _, roomID := range roomIDs {
			rm.checkRoom(clubID, roomID)
			roomsChecked++
		}
	}

	// Проверяем, не было ли ошибок при сканировании
	if err := iter.Err(); err != nil {
		rm.logger.Errorf("Ошибка при сканировании клубов: %v", err)
	}

	// Логируем статистику (только если проверили хотя бы одну комнату)
	if roomsChecked > 0 {
		rm.logger.Debugf("Проверено комнат: %d", roomsChecked)
	}
}

// checkRoom - проверяет состояние конкретной комнаты
func (rm *RoomMonitor) checkRoom(clubID, roomID string) {
	// Получаем количество игроков
	playersCount, err := rm.gameStateService.GetPlayersCount(clubID, roomID)
	if err != nil {
		rm.logger.Errorf("Ошибка при получении количества игроков в комнате %s:%s: %v", clubID, roomID, err)
		return
	}

	// Получаем текущее состояние игры
	game, err := rm.gameStateService.GetGameState(clubID, roomID)
	if err != nil {
		rm.logger.Errorf("Ошибка при получении состояния игры %s:%s: %v", clubID, roomID, err)
		return
	}

	// Если нет данных об игре, пропускаем
	if game == nil {
		return
	}

	currentPhase := game.Phase

	// === ЛОГИКА УПРАВЛЕНИЯ СОСТОЯНИЕМ ИГРЫ ===

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

	// СЛУЧАЙ 3: Игра идет нормально -> просто логируем (опционально)
	if playersCount >= int64(rm.config.MinPlayersToStart) && currentPhase != "waiting" {
		// Можно раскомментировать для детального логирования
		// rm.logger.GameInProgress(clubID, roomID, string(currentPhase), int(playersCount))
	}
}

// handleGameStart - внутренняя логика запуска игры (без импорта handlers)
func (rm *RoomMonitor) handleGameStart(clubID, roomID string, playersCount int) error {
	// Логика из GameStartHandler перенесена сюда
	rm.logger.Infof("Запуск игры в комнате %s:%s", clubID, roomID)

	// Генерируем ID игры
	gameID := utils.GetCurrentTime().Format("20060102150405") + "_" + clubID + "_" + roomID

	// Текущее время
	startedAt := utils.GetISO8601Time()

	// Обновляем Redis
	pipe := rm.redis.Pipeline()
	roomInfoKey := rm.redis.GetKeys().RoomInfo(clubID, roomID)
	gameStateKey := rm.redis.GetKeys().GameState(clubID, roomID)

	pipe.HSet(rm.redis.GetClient().Context(), roomInfoKey, "status", "gaming")
	pipe.HSet(rm.redis.GetClient().Context(), gameStateKey, "phase", "pre_flop")
	pipe.HSet(rm.redis.GetClient().Context(), gameStateKey, "game_id", gameID)
	pipe.HSet(rm.redis.GetClient().Context(), gameStateKey, "started_at", startedAt)
	pipe.HSet(rm.redis.GetClient().Context(), gameStateKey, "pot", 0)
	pipe.HSet(rm.redis.GetClient().Context(), gameStateKey, "current_bet", 0)

	_, err := pipe.Exec(rm.redis.GetClient().Context())
	if err != nil {
		return err
	}

	// Логируем
	rm.actionLogger.LogGameStarted(clubID, roomID, gameID, playersCount)
	rm.logger.GameStarted(clubID, roomID, gameID, playersCount)

	return nil
}

// handleGameStop - внутренняя логика остановки игры (без импорта handlers)
func (rm *RoomMonitor) handleGameStop(clubID, roomID, previousPhase, reason string) error {
	// Логика из GameStopHandler перенесена сюда
	rm.logger.Infof("Остановка игры в комнате %s:%s", clubID, roomID)

	// Обновляем Redis
	pipe := rm.redis.Pipeline()
	roomInfoKey := rm.redis.GetKeys().RoomInfo(clubID, roomID)
	gameStateKey := rm.redis.GetKeys().GameState(clubID, roomID)

	pipe.HSet(rm.redis.GetClient().Context(), roomInfoKey, "status", "waiting")
	pipe.HSet(rm.redis.GetClient().Context(), gameStateKey, "phase", "waiting")
	pipe.HSet(rm.redis.GetClient().Context(), gameStateKey, "game_id", "")
	pipe.HSet(rm.redis.GetClient().Context(), gameStateKey, "started_at", "")
	pipe.HSet(rm.redis.GetClient().Context(), gameStateKey, "pot", 0)
	pipe.HSet(rm.redis.GetClient().Context(), gameStateKey, "current_bet", 0)
	pipe.HSet(rm.redis.GetClient().Context(), gameStateKey, "community_cards", "[]")

	_, err := pipe.Exec(rm.redis.GetClient().Context())
	if err != nil {
		return err
	}

	// Логируем
	rm.actionLogger.LogGameStopped(clubID, roomID, previousPhase, reason)
	rm.logger.GameStopped(clubID, roomID, previousPhase, reason)

	return nil
}

// GetStatistics - возвращает статистику мониторинга
func (rm *RoomMonitor) GetStatistics() map[string]interface{} {
	stats := map[string]interface{}{
		"is_running":      rm.isRunning,
		"check_interval":  rm.config.CheckInterval.String(),
		"min_players":     rm.config.MinPlayersToStart,
		"redis_connected": rm.redis.IsConnected(),
	}

	return stats
}

// HealthCheck - проверяет здоровье мониторинга
func (rm *RoomMonitor) HealthCheck() error {
	// Проверяем, что мониторинг запущен
	if !rm.isRunning {
		return ErrMonitorNotRunning
	}

	// Проверяем подключение к Redis
	if err := rm.redis.HealthCheck(); err != nil {
		return err
	}

	return nil
}

// ForceCheck - принудительно запускает проверку всех комнат (вне расписания)
func (rm *RoomMonitor) ForceCheck() {
	rm.logger.Info("Принудительная проверка всех комнат...")
	rm.checkAllRooms()
	rm.logger.Success("Принудительная проверка завершена")
}

// CheckSpecificRoom - проверяет конкретную комнату (вне общего цикла)
func (rm *RoomMonitor) CheckSpecificRoom(clubID, roomID string) error {
	rm.logger.Infof("Проверка комнаты %s:%s...", clubID, roomID)

	// Проверяем, существует ли комната
	exists, err := rm.gameStateService.RoomExists(clubID, roomID)
	if err != nil {
		return err
	}

	if !exists {
		rm.logger.Warningf("Комната %s:%s не существует", clubID, roomID)
		return ErrRoomNotFound
	}

	// Проверяем комнату
	rm.checkRoom(clubID, roomID)

	rm.logger.Successf("Проверка комнаты %s:%s завершена", clubID, roomID)
	return nil
}

// === ОШИБКИ ===

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
