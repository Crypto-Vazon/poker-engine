package services

import (
	"encoding/json"
	"poker-engine/storage"
	"poker-engine/utils"
)

// ActionLogger - сервис для записи действий в историю комнаты
type ActionLogger struct {
	// redis - клиент для работы с Redis
	redis *storage.RedisClient

	// logger - логгер для вывода сообщений
	logger *utils.Logger
}

// NewActionLogger - создает новый экземпляр ActionLogger
func NewActionLogger(redis *storage.RedisClient) *ActionLogger {
	return &ActionLogger{
		redis:  redis,
		logger: utils.NewLogger("ActionLogger"),
	}
}

// ActionData - структура для данных действия
type ActionData struct {
	// Тип действия (game_started, game_stopped, player_joined и т.д.)
	Action string `json:"action"`

	// Временная метка (Unix timestamp в секундах)
	Timestamp int64 `json:"timestamp"`

	// Источник действия (game_engine, player, system и т.д.)
	Source string `json:"source"`

	// Дополнительные данные (зависят от типа действия)
	Data map[string]interface{} `json:"data,omitempty"`
}

// LogAction - записывает действие в историю комнаты
// Параметры:
//   - clubID: ID клуба
//   - roomID: ID комнаты
//   - action: тип действия (например: "game_started", "player_joined")
//   - data: дополнительные данные для действия (может быть nil)
func (al *ActionLogger) LogAction(clubID, roomID, action string, data map[string]interface{}) error {
	// Создаем структуру действия
	actionData := ActionData{
		Action:    action,
		Timestamp: utils.GetCurrentTimestamp(),
		Source:    "game_engine", // Все действия от движка имеют этот источник
		Data:      data,
	}

	// Конвертируем в JSON
	jsonData, err := json.Marshal(actionData)
	if err != nil {
		al.logger.Errorf("Ошибка при сериализации действия '%s' в JSON: %v", action, err)
		return err
	}

	// Получаем ключ для списка действий комнаты
	actionsKey := al.redis.GetKeys().RoomActions(clubID, roomID)

	// Добавляем действие в конец списка (RPUSH)
	err = al.redis.RPush(actionsKey, string(jsonData))
	if err != nil {
		al.logger.Errorf("Ошибка при записи действия '%s' в Redis: %v", action, err)
		return err
	}

	// Логируем успешную запись (только в debug режиме, чтобы не засорять логи)
	// al.logger.Debugf("Действие '%s' записано для комнаты %s:%s", action, clubID, roomID)

	return nil
}

// LogGameStarted - записывает действие запуска игры
func (al *ActionLogger) LogGameStarted(clubID, roomID, gameID string, playersCount int) error {
	return al.LogAction(clubID, roomID, "game_started", map[string]interface{}{
		"game_id":       gameID,
		"phase":         "pre_flop",
		"players_count": playersCount,
	})
}

// LogGameStopped - записывает действие остановки игры
func (al *ActionLogger) LogGameStopped(clubID, roomID, previousPhase, reason string) error {
	return al.LogAction(clubID, roomID, "game_stopped", map[string]interface{}{
		"previous_phase": previousPhase,
		"reason":         reason,
	})
}

// LogPhaseChanged - записывает действие смены фазы игры
func (al *ActionLogger) LogPhaseChanged(clubID, roomID, oldPhase, newPhase string) error {
	return al.LogAction(clubID, roomID, "phase_changed", map[string]interface{}{
		"old_phase": oldPhase,
		"new_phase": newPhase,
	})
}

// LogPlayerJoined - записывает действие присоединения игрока к комнате
func (al *ActionLogger) LogPlayerJoined(clubID, roomID, userID string, role string) error {
	return al.LogAction(clubID, roomID, "player_joined", map[string]interface{}{
		"user_id": userID,
		"role":    role, // "player" или "spectator"
	})
}

// LogPlayerLeft - записывает действие выхода игрока из комнаты
func (al *ActionLogger) LogPlayerLeft(clubID, roomID, userID string, role string) error {
	return al.LogAction(clubID, roomID, "player_left", map[string]interface{}{
		"user_id": userID,
		"role":    role,
	})
}

// LogPlayerSatDown - записывает действие, когда игрок садится за стол
func (al *ActionLogger) LogPlayerSatDown(clubID, roomID, userID string, position int, buyIn int) error {
	return al.LogAction(clubID, roomID, "player_sat_down", map[string]interface{}{
		"user_id":  userID,
		"position": position,
		"buy_in":   buyIn,
	})
}

// LogPlayerStoodUp - записывает действие, когда игрок встает из-за стола
func (al *ActionLogger) LogPlayerStoodUp(clubID, roomID, userID string, position int, chipsLeft int) error {
	return al.LogAction(clubID, roomID, "player_stood_up", map[string]interface{}{
		"user_id":    userID,
		"position":   position,
		"chips_left": chipsLeft,
	})
}

// LogPlayerAction - записывает игровое действие игрока (fold, call, raise и т.д.)
func (al *ActionLogger) LogPlayerAction(clubID, roomID, userID, action string, amount int) error {
	data := map[string]interface{}{
		"user_id": userID,
		"action":  action,
	}

	// Добавляем сумму только если она больше 0
	if amount > 0 {
		data["amount"] = amount
	}

	return al.LogAction(clubID, roomID, "player_action", data)
}

// LogDealerMoved - записывает действие перемещения дилера
func (al *ActionLogger) LogDealerMoved(clubID, roomID string, oldPosition, newPosition int) error {
	return al.LogAction(clubID, roomID, "dealer_moved", map[string]interface{}{
		"old_position": oldPosition,
		"new_position": newPosition,
	})
}

// LogBlindsPosted - записывает действие размещения блайндов
func (al *ActionLogger) LogBlindsPosted(clubID, roomID string, smallBlindUser, bigBlindUser string, smallBlindAmount, bigBlindAmount int) error {
	return al.LogAction(clubID, roomID, "blinds_posted", map[string]interface{}{
		"small_blind_user":   smallBlindUser,
		"big_blind_user":     bigBlindUser,
		"small_blind_amount": smallBlindAmount,
		"big_blind_amount":   bigBlindAmount,
	})
}

// LogCardsDealt - записывает действие раздачи карт
func (al *ActionLogger) LogCardsDealt(clubID, roomID string, playersCount int) error {
	return al.LogAction(clubID, roomID, "cards_dealt", map[string]interface{}{
		"players_count": playersCount,
	})
}

// LogCommunityCardsRevealed - записывает действие открытия общих карт
func (al *ActionLogger) LogCommunityCardsRevealed(clubID, roomID, phase string, cardsCount int) error {
	return al.LogAction(clubID, roomID, "community_cards_revealed", map[string]interface{}{
		"phase":       phase,
		"cards_count": cardsCount,
	})
}

// LogPotAwarded - записывает действие присуждения банка
func (al *ActionLogger) LogPotAwarded(clubID, roomID, winnerUserID string, amount int) error {
	return al.LogAction(clubID, roomID, "pot_awarded", map[string]interface{}{
		"winner_user_id": winnerUserID,
		"amount":         amount,
	})
}

// LogRoundFinished - записывает действие завершения раунда
func (al *ActionLogger) LogRoundFinished(clubID, roomID string, roundNumber int, totalPot int) error {
	return al.LogAction(clubID, roomID, "round_finished", map[string]interface{}{
		"round_number": roundNumber,
		"total_pot":    totalPot,
	})
}

// LogError - записывает ошибку, произошедшую в комнате
func (al *ActionLogger) LogError(clubID, roomID, errorType, errorMessage string) error {
	return al.LogAction(clubID, roomID, "error", map[string]interface{}{
		"error_type":    errorType,
		"error_message": errorMessage,
	})
}

// === МЕТОДЫ ДЛЯ ЧТЕНИЯ ИСТОРИИ ===

// GetRecentActions - получает последние N действий из истории комнаты
// Параметры:
//   - clubID: ID клуба
//   - roomID: ID комнаты
//   - count: количество последних действий (по умолчанию 100)
//
// Возвращает массив действий в хронологическом порядке (от старых к новым)
func (al *ActionLogger) GetRecentActions(clubID, roomID string, count int64) ([]ActionData, error) {
	if count <= 0 {
		count = 100 // По умолчанию последние 100 действий
	}

	// Получаем ключ для списка действий
	actionsKey := al.redis.GetKeys().RoomActions(clubID, roomID)

	// Получаем последние N элементов из списка
	// LRANGE с отрицательными индексами: -count означает "count элементов с конца"
	jsonStrings, err := al.redis.LRange(actionsKey, -count, -1)
	if err != nil {
		al.logger.Errorf("Ошибка при чтении истории действий комнаты %s:%s: %v", clubID, roomID, err)
		return nil, err
	}

	// Парсим JSON строки в структуры ActionData
	actions := make([]ActionData, 0, len(jsonStrings))
	for _, jsonStr := range jsonStrings {
		var action ActionData
		if err := json.Unmarshal([]byte(jsonStr), &action); err != nil {
			al.logger.Warningf("Ошибка при парсинге действия из JSON: %v", err)
			continue // Пропускаем некорректные записи
		}
		actions = append(actions, action)
	}

	return actions, nil
}

// GetActionsCount - возвращает общее количество действий в истории комнаты
func (al *ActionLogger) GetActionsCount(clubID, roomID string) (int64, error) {
	actionsKey := al.redis.GetKeys().RoomActions(clubID, roomID)
	return al.redis.LLen(actionsKey)
}

// ClearHistory - очищает всю историю действий комнаты
// ВНИМАНИЕ: Используйте с осторожностью! Это удалит ВСЮ историю.
func (al *ActionLogger) ClearHistory(clubID, roomID string) error {
	actionsKey := al.redis.GetKeys().RoomActions(clubID, roomID)
	err := al.redis.Del(actionsKey)
	if err != nil {
		al.logger.Errorf("Ошибка при очистке истории комнаты %s:%s: %v", clubID, roomID, err)
		return err
	}

	al.logger.Infof("История действий комнаты %s:%s очищена", clubID, roomID)
	return nil
}

// TrimHistory - обрезает историю, оставляя только последние N действий
// Полезно для ограничения размера истории в Redis
func (al *ActionLogger) TrimHistory(clubID, roomID string, keepLast int64) error {
	if keepLast <= 0 {
		return al.ClearHistory(clubID, roomID)
	}

	actionsKey := al.redis.GetKeys().RoomActions(clubID, roomID)

	// LTRIM оставляет только элементы в указанном диапазоне
	// -keepLast означает "последние keepLast элементов"
	err := al.redis.GetClient().LTrim(al.redis.GetClient().Context(), actionsKey, -keepLast, -1).Err()
	if err != nil {
		al.logger.Errorf("Ошибка при обрезке истории комнаты %s:%s: %v", clubID, roomID, err)
		return err
	}

	al.logger.Infof("История комнаты %s:%s обрезана, оставлено последних %d действий", clubID, roomID, keepLast)
	return nil
}

// === ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ ===

// GetActionsByType - получает все действия определенного типа
// Например: все "game_started" или все "player_action"
func (al *ActionLogger) GetActionsByType(clubID, roomID, actionType string, limit int64) ([]ActionData, error) {
	// Получаем все последние действия
	allActions, err := al.GetRecentActions(clubID, roomID, limit)
	if err != nil {
		return nil, err
	}

	// Фильтруем по типу
	filtered := make([]ActionData, 0)
	for _, action := range allActions {
		if action.Action == actionType {
			filtered = append(filtered, action)
		}
	}

	return filtered, nil
}

// GetActionsByUser - получает все действия конкретного пользователя
func (al *ActionLogger) GetActionsByUser(clubID, roomID, userID string, limit int64) ([]ActionData, error) {
	// Получаем все последние действия
	allActions, err := al.GetRecentActions(clubID, roomID, limit)
	if err != nil {
		return nil, err
	}

	// Фильтруем по пользователю
	filtered := make([]ActionData, 0)
	for _, action := range allActions {
		// Проверяем, есть ли user_id в дополнительных данных
		if action.Data != nil {
			if uid, ok := action.Data["user_id"].(string); ok && uid == userID {
				filtered = append(filtered, action)
			}
		}
	}

	return filtered, nil
}
