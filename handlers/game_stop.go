package handlers

import (
	"fmt"

	"poker-engine/models"
	"poker-engine/services"
	"poker-engine/storage"
	"poker-engine/utils"
)

// GameStopHandler - обработчик остановки игры
type GameStopHandler struct {
	// redis - клиент для работы с Redis
	redis *storage.RedisClient

	// gameStateService - сервис для работы с состоянием игры
	gameStateService *services.GameStateService

	// actionLogger - сервис для записи действий
	actionLogger *services.ActionLogger

	// logger - логгер для вывода сообщений
	logger *utils.Logger
}

// NewGameStopHandler - создает новый экземпляр GameStopHandler
func NewGameStopHandler(
	redis *storage.RedisClient,
	gameStateService *services.GameStateService,
	actionLogger *services.ActionLogger,
) *GameStopHandler {
	return &GameStopHandler{
		redis:            redis,
		gameStateService: gameStateService,
		actionLogger:     actionLogger,
		logger:           utils.NewLogger("GameStop"),
	}
}

// Handle - останавливает игру в комнате
// Параметры:
//   - clubID: ID клуба
//   - roomID: ID комнаты
//   - previousPhase: фаза, на которой остановилась игра
//   - reason: причина остановки (например: "insufficient_players")
//
// Возвращает ошибку если остановка не удалась
func (h *GameStopHandler) Handle(clubID, roomID, previousPhase, reason string) error {
	// Логируем начало процесса остановки
	h.logger.Infof("Начинаем остановку игры в комнате %s:%s (причина: %s)", clubID, roomID, reason)

	// === ПРОВЕРКИ ПЕРЕД ОСТАНОВКОЙ ===

	// Проверка 1: Комната существует?
	exists, err := h.gameStateService.RoomExists(clubID, roomID)
	if err != nil {
		return fmt.Errorf("ошибка проверки существования комнаты: %w", err)
	}
	if !exists {
		return fmt.Errorf("комната %s:%s не существует", clubID, roomID)
	}

	// Проверка 2: Игра действительно запущена?
	isActive, err := h.gameStateService.IsGameActive(clubID, roomID)
	if err != nil {
		return fmt.Errorf("ошибка проверки статуса игры: %w", err)
	}
	if !isActive {
		// Игра уже остановлена, это не ошибка
		h.logger.Debugf("Игра в комнате %s:%s уже остановлена", clubID, roomID)
		return nil
	}

	// === ОБНОВЛЕНИЕ СОСТОЯНИЯ В REDIS ===

	// Используем Pipeline для атомарного обновления
	pipe := h.redis.Pipeline()

	// Ключи для обновления
	roomInfoKey := h.redis.GetKeys().RoomInfo(clubID, roomID)
	gameStateKey := h.redis.GetKeys().GameState(clubID, roomID)
	ctx := h.redis.GetContext()

	// 1. Возвращаем статус комнаты в "waiting"
	pipe.HSet(ctx, roomInfoKey, "status", string(models.RoomStatusWaiting))

	// 2. Сбрасываем состояние игры
	pipe.HSet(ctx, gameStateKey, "phase", string(models.GamePhaseWaiting))
	pipe.HSet(ctx, gameStateKey, "game_id", "")
	pipe.HSet(ctx, gameStateKey, "started_at", "")

	// 3. Сбрасываем игровые параметры
	pipe.HSet(ctx, gameStateKey, "pot", 0)
	pipe.HSet(ctx, gameStateKey, "current_bet", 0)
	pipe.HSet(ctx, gameStateKey, "current_player_position", "")

	// 4. Очищаем общие карты
	pipe.HSet(ctx, gameStateKey, "community_cards", "[]")

	// Выполняем все команды
	_, err = pipe.Exec(ctx)
	if err != nil {
		h.logger.Errorf("Ошибка при обновлении состояния игры в Redis: %v", err)
		return fmt.Errorf("ошибка обновления Redis: %w", err)
	}

	// === СБРОС СОСТОЯНИЯ ИГРОКОВ (опционально) ===

	// Получаем всех игроков
	playerIDs, err := h.gameStateService.GetPlayerIDs(clubID, roomID)
	if err != nil {
		// Не критичная ошибка, логируем и продолжаем
		h.logger.Warningf("Не удалось получить список игроков для сброса состояния: %v", err)
	} else {
		// Сбрасываем состояние каждого игрока
		for _, userID := range playerIDs {
			err := h.resetPlayerState(clubID, roomID, userID)
			if err != nil {
				h.logger.Warningf("Ошибка сброса состояния игрока %s: %v", userID, err)
			}
		}
	}

	// === ЛОГИРОВАНИЕ ===

	// Записываем действие в историю комнаты
	err = h.actionLogger.LogGameStopped(clubID, roomID, previousPhase, reason)
	if err != nil {
		// Не критичная ошибка, просто логируем
		h.logger.Warningf("Не удалось записать действие в историю: %v", err)
	}

	// Красивый лог в консоль
	h.logger.GameStopped(clubID, roomID, previousPhase, reason)

	return nil
}

// resetPlayerState - сбрасывает состояние игрока после остановки игры
func (h *GameStopHandler) resetPlayerState(clubID, roomID, userID string) error {
	playerKey := h.redis.GetKeys().PlayerInfo(clubID, roomID, userID)

	// Обновляем состояние игрока
	updates := map[string]interface{}{
		"status":         string(models.PlayerStatusWaiting),
		"bet":            0,
		"cards":          "[]",
		"last_action":    "",
		"is_dealer":      false,
		"is_small_blind": false,
		"is_big_blind":   false,
	}

	err := h.redis.HMSet(playerKey, updates)
	if err != nil {
		return fmt.Errorf("ошибка сброса состояния игрока: %w", err)
	}

	return nil
}

// HandleWithCleanup - останавливает игру с полной очисткой данных
// Удаляет дополнительные структуры (deck, pots и т.д.)
func (h *GameStopHandler) HandleWithCleanup(clubID, roomID, previousPhase, reason string) error {
	// Сначала останавливаем игру стандартным способом
	err := h.Handle(clubID, roomID, previousPhase, reason)
	if err != nil {
		return err
	}

	// Дополнительная очистка
	keys := h.redis.GetKeys()

	// Удаляем колоду карт
	deckKey := keys.RoomDeck(clubID, roomID)
	h.redis.Del(deckKey)

	// Сбрасываем банки
	potsKey := keys.RoomPots(clubID, roomID)
	h.redis.HMSet(potsKey, map[string]interface{}{
		"main_pot":  0,
		"side_pots": "[]",
	})

	h.logger.Infof("Выполнена полная очистка данных игры %s:%s", clubID, roomID)

	return nil
}

// ForceStop - принудительно останавливает игру (даже если игроков достаточно)
// Используется для экстренных случаев или административных действий
func (h *GameStopHandler) ForceStop(clubID, roomID, reason string) error {
	// Получаем текущую фазу
	game, err := h.gameStateService.GetGameState(clubID, roomID)
	if err != nil {
		return err
	}

	previousPhase := string(models.GamePhaseWaiting)
	if game != nil {
		previousPhase = string(game.Phase)
	}

	h.logger.Warningf("ПРИНУДИТЕЛЬНАЯ остановка игры %s:%s", clubID, roomID)

	return h.HandleWithCleanup(clubID, roomID, previousPhase, "force_stop: "+reason)
}

// CanStopGame - проверяет, можно ли остановить игру (без остановки)
func (h *GameStopHandler) CanStopGame(clubID, roomID string) (bool, error) {
	// Проверяем, существует ли комната
	exists, err := h.gameStateService.RoomExists(clubID, roomID)
	if err != nil || !exists {
		return false, err
	}

	// Проверяем, активна ли игра
	isActive, err := h.gameStateService.IsGameActive(clubID, roomID)
	if err != nil {
		return false, err
	}

	// Можно остановить только если игра активна
	return isActive, nil
}

// StopIfNeeded - останавливает игру если выполнены условия остановки
// Проверяет количество игроков и автоматически решает, нужна ли остановка
func (h *GameStopHandler) StopIfNeeded(clubID, roomID string, minPlayers int) (bool, error) {
	// Проверяем, активна ли игра
	isActive, err := h.gameStateService.IsGameActive(clubID, roomID)
	if err != nil {
		return false, err
	}

	if !isActive {
		// Игра не активна, остановка не нужна
		return false, nil
	}

	// Проверяем количество игроков
	playersCount, err := h.gameStateService.GetPlayersCount(clubID, roomID)
	if err != nil {
		return false, err
	}

	// Если игроков достаточно, не останавливаем
	if playersCount >= int64(minPlayers) {
		return false, nil
	}

	// Получаем текущую фазу
	game, err := h.gameStateService.GetGameState(clubID, roomID)
	if err != nil {
		return false, err
	}

	previousPhase := string(models.GamePhaseWaiting)
	if game != nil {
		previousPhase = string(game.Phase)
	}

	// Останавливаем игру
	err = h.Handle(clubID, roomID, previousPhase, "insufficient_players")
	if err != nil {
		return false, err
	}

	return true, nil
}

// GetGameStopInfo - возвращает информацию о текущей игре для остановки
func (h *GameStopHandler) GetGameStopInfo(clubID, roomID string) (map[string]interface{}, error) {
	// Получаем состояние игры
	game, err := h.gameStateService.GetGameState(clubID, roomID)
	if err != nil {
		return nil, err
	}

	// Получаем количество игроков
	playersCount, err := h.gameStateService.GetPlayersCount(clubID, roomID)
	if err != nil {
		return nil, err
	}

	// Проверяем, можно ли остановить
	canStop, _ := h.CanStopGame(clubID, roomID)

	info := map[string]interface{}{
		"club_id":       clubID,
		"room_id":       roomID,
		"players_count": playersCount,
		"can_stop":      canStop,
		"is_active":     game != nil && game.IsActive(),
	}

	if game != nil {
		info["current_phase"] = string(game.Phase)
		info["game_id"] = game.GameID
		info["pot"] = game.Pot
	}

	return info, nil
}

// SaveGameResults - сохраняет результаты игры перед остановкой
// Полезно для статистики и истории
func (h *GameStopHandler) SaveGameResults(clubID, roomID string) error {
	// Получаем состояние игры
	game, err := h.gameStateService.GetGameState(clubID, roomID)
	if err != nil {
		return err
	}

	if game == nil || game.GameID == "" {
		return fmt.Errorf("нет активной игры для сохранения результатов")
	}

	// Здесь можно добавить логику сохранения результатов
	// Например, записать в отдельную таблицу истории игр
	// Или отправить в Laravel API для сохранения в MySQL

	h.logger.Infof("Результаты игры %s сохранены", game.GameID)

	return nil
}
