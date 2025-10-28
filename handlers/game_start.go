package handlers

import (
	"fmt"
	"time"

	"poker-engine/models"
	"poker-engine/services"
	"poker-engine/storage"
	"poker-engine/utils"
)

// GameStartHandler - обработчик запуска игры
type GameStartHandler struct {
	// redis - клиент для работы с Redis
	redis *storage.RedisClient

	// gameStateService - сервис для работы с состоянием игры
	gameStateService *services.GameStateService

	// actionLogger - сервис для записи действий
	actionLogger *services.ActionLogger

	// logger - логгер для вывода сообщений
	logger *utils.Logger
}

// NewGameStartHandler - создает новый экземпляр GameStartHandler
func NewGameStartHandler(
	redis *storage.RedisClient,
	gameStateService *services.GameStateService,
	actionLogger *services.ActionLogger,
) *GameStartHandler {
	return &GameStartHandler{
		redis:            redis,
		gameStateService: gameStateService,
		actionLogger:     actionLogger,
		logger:           utils.NewLogger("GameStart"),
	}
}

// Handle - запускает игру в комнате
// Параметры:
//   - clubID: ID клуба
//   - roomID: ID комнаты
//   - playersCount: текущее количество игроков
//
// Возвращает ошибку если запуск не удался
func (h *GameStartHandler) Handle(clubID, roomID string, playersCount int) error {
	// Логируем начало процесса запуска
	h.logger.Infof("Начинаем запуск игры в комнате %s:%s (игроков: %d)", clubID, roomID, playersCount)

	// === ПРОВЕРКИ ПЕРЕД ЗАПУСКОМ ===

	// Проверка 1: Комната существует?
	exists, err := h.gameStateService.RoomExists(clubID, roomID)
	if err != nil {
		return fmt.Errorf("ошибка проверки существования комнаты: %w", err)
	}
	if !exists {
		return fmt.Errorf("комната %s:%s не существует", clubID, roomID)
	}

	// Проверка 2: Игра уже не запущена?
	isActive, err := h.gameStateService.IsGameActive(clubID, roomID)
	if err != nil {
		return fmt.Errorf("ошибка проверки статуса игры: %w", err)
	}
	if isActive {
		h.logger.Warningf("Игра в комнате %s:%s уже запущена", clubID, roomID)
		return nil // Не ошибка, просто игнорируем
	}

	// Проверка 3: Достаточно игроков?
	actualPlayersCount, err := h.gameStateService.GetPlayersCount(clubID, roomID)
	if err != nil {
		return fmt.Errorf("ошибка получения количества игроков: %w", err)
	}
	if actualPlayersCount < 2 {
		return fmt.Errorf("недостаточно игроков для запуска (нужно минимум 2, есть %d)", actualPlayersCount)
	}

	// === ПОДГОТОВКА ДАННЫХ ===

	// Генерируем уникальный ID игры
	// Формат: game_{clubId}_{roomId}_{timestamp}
	gameID := fmt.Sprintf("game_%s_%s_%d", clubID, roomID, time.Now().Unix())

	// Текущее время в ISO 8601 формате
	startedAt := utils.GetISO8601Time()

	// === ОБНОВЛЕНИЕ СОСТОЯНИЯ В REDIS ===

	// Используем Pipeline для атомарного обновления всех данных
	pipe := h.redis.Pipeline()

	// Ключи для обновления
	roomInfoKey := h.redis.GetKeys().RoomInfo(clubID, roomID)
	gameStateKey := h.redis.GetKeys().GameState(clubID, roomID)

	// 1. Обновляем статус комнаты на "gaming"
	pipe.HSet(h.redis.GetClient().Context(), roomInfoKey, "status", string(models.RoomStatusGaming))

	// 2. Обновляем состояние игры
	pipe.HSet(h.redis.GetClient().Context(), gameStateKey, "phase", string(models.GamePhasePreFlop))
	pipe.HSet(h.redis.GetClient().Context(), gameStateKey, "game_id", gameID)
	pipe.HSet(h.redis.GetClient().Context(), gameStateKey, "started_at", startedAt)

	// Опционально: можно сбросить pot и current_bet (если они не были сброшены ранее)
	pipe.HSet(h.redis.GetClient().Context(), gameStateKey, "pot", 0)
	pipe.HSet(h.redis.GetClient().Context(), gameStateKey, "current_bet", 0)

	// Выполняем все команды
	_, err = pipe.Exec(h.redis.GetClient().Context())
	if err != nil {
		h.logger.Errorf("Ошибка при обновлении состояния игры в Redis: %v", err)
		return fmt.Errorf("ошибка обновления Redis: %w", err)
	}

	// === ЛОГИРОВАНИЕ ===

	// Записываем действие в историю комнаты
	err = h.actionLogger.LogGameStarted(clubID, roomID, gameID, playersCount)
	if err != nil {
		// Не критичная ошибка, просто логируем
		h.logger.Warningf("Не удалось записать действие в историю: %v", err)
	}

	// Красивый лог в консоль
	h.logger.GameStarted(clubID, roomID, gameID, playersCount)

	return nil
}

// HandleWithValidation - запускает игру с дополнительной валидацией игроков
// Проверяет, что все игроки в состоянии "waiting" и готовы к игре
func (h *GameStartHandler) HandleWithValidation(clubID, roomID string) error {
	// Получаем всех игроков
	playerIDs, err := h.gameStateService.GetPlayerIDs(clubID, roomID)
	if err != nil {
		return fmt.Errorf("ошибка получения списка игроков: %w", err)
	}

	if len(playerIDs) < 2 {
		return fmt.Errorf("недостаточно игроков для запуска")
	}

	// Проверяем статус каждого игрока
	validPlayersCount := 0
	for _, userID := range playerIDs {
		player, err := h.gameStateService.GetPlayer(clubID, roomID, userID)
		if err != nil {
			h.logger.Warningf("Ошибка получения данных игрока %s: %v", userID, err)
			continue
		}

		// Игрок должен иметь фишки и не быть в sit_out
		if player != nil && player.HasChips() && !player.IsSittingOut() {
			validPlayersCount++
		}
	}

	if validPlayersCount < 2 {
		return fmt.Errorf("недостаточно готовых игроков (нужно 2, готовых %d)", validPlayersCount)
	}

	// Запускаем игру
	return h.Handle(clubID, roomID, validPlayersCount)
}

// CanStartGame - проверяет, можно ли запустить игру (без запуска)
// Возвращает true если все условия выполнены
func (h *GameStartHandler) CanStartGame(clubID, roomID string, minPlayers int) (bool, error) {
	// Проверяем существование комнаты
	exists, err := h.gameStateService.RoomExists(clubID, roomID)
	if err != nil || !exists {
		return false, err
	}

	// Проверяем, что игра не активна
	isActive, err := h.gameStateService.IsGameActive(clubID, roomID)
	if err != nil {
		return false, err
	}
	if isActive {
		return false, nil
	}

	// Проверяем количество игроков
	playersCount, err := h.gameStateService.GetPlayersCount(clubID, roomID)
	if err != nil {
		return false, err
	}

	return playersCount >= int64(minPlayers), nil
}

// PrepareGameStart - подготавливает данные для запуска игры
// Устанавливает начальные позиции дилера, блайндов и т.д.
// Это можно вызвать перед Handle() для более детальной настройки
func (h *GameStartHandler) PrepareGameStart(clubID, roomID string) error {
	// Получаем всех игроков
	playerIDs, err := h.gameStateService.GetPlayerIDs(clubID, roomID)
	if err != nil {
		return err
	}

	if len(playerIDs) < 2 {
		return fmt.Errorf("недостаточно игроков")
	}

	// Получаем текущую информацию об игре
	game, err := h.gameStateService.GetGameState(clubID, roomID)
	if err != nil {
		return err
	}

	// Устанавливаем начальную позицию дилера (обычно 0)
	// В будущем можно сделать ротацию
	dealerPosition := 0
	if game != nil && game.DealerPosition >= 0 {
		// Если игра уже была, берем следующую позицию
		dealerPosition = (game.DealerPosition + 1) % len(playerIDs)
	}

	// Обновляем позицию дилера
	updates := map[string]interface{}{
		"dealer_position": dealerPosition,
	}

	err = h.gameStateService.UpdateGameState(clubID, roomID, updates)
	if err != nil {
		return fmt.Errorf("ошибка обновления позиции дилера: %w", err)
	}

	h.logger.Infof("Дилер установлен на позицию %d для комнаты %s:%s", dealerPosition, clubID, roomID)

	return nil
}

// GetGameStartInfo - возвращает информацию о предстоящем запуске игры
// Полезно для отображения на фронтенде перед стартом
func (h *GameStartHandler) GetGameStartInfo(clubID, roomID string) (map[string]interface{}, error) {
	// Получаем информацию о комнате
	room, err := h.gameStateService.GetRoomInfo(clubID, roomID)
	if err != nil {
		return nil, err
	}

	// Получаем количество игроков
	playersCount, err := h.gameStateService.GetPlayersCount(clubID, roomID)
	if err != nil {
		return nil, err
	}

	// Проверяем, можно ли начать игру
	canStart, _ := h.CanStartGame(clubID, roomID, 2)

	info := map[string]interface{}{
		"club_id":        clubID,
		"room_id":        roomID,
		"players_count":  playersCount,
		"max_players":    room.MaxPlayers,
		"can_start":      canStart,
		"small_blind":    room.SmallBlind,
		"big_blind":      room.BigBlind,
		"current_status": room.Status,
	}

	return info, nil
}
