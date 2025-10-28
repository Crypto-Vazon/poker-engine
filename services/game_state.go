package services

import (
	"poker-engine/models"
	"poker-engine/storage"
	"poker-engine/utils"
)

// GameStateService - сервис для работы с состоянием игры
type GameStateService struct {
	// redis - клиент для работы с Redis
	redis *storage.RedisClient

	// logger - логгер для вывода сообщений
	logger *utils.Logger
}

// NewGameStateService - создает новый экземпляр GameStateService
func NewGameStateService(redis *storage.RedisClient) *GameStateService {
	return &GameStateService{
		redis:  redis,
		logger: utils.NewLogger("GameState"),
	}
}

// === МЕТОДЫ ДЛЯ ЧТЕНИЯ СОСТОЯНИЯ ===

// GetGameState - получает состояние игры из Redis
// Возвращает nil если игра не найдена
func (gs *GameStateService) GetGameState(clubID, roomID string) (*models.Game, error) {
	// Получаем ключ для состояния игры
	gameKey := gs.redis.GetKeys().GameState(clubID, roomID)

	// Читаем все поля hash
	data, err := gs.redis.HGetAll(gameKey)
	if err != nil {
		gs.logger.Errorf("Ошибка при получении состояния игры %s:%s: %v", clubID, roomID, err)
		return nil, err
	}

	// Проверяем, что данные существуют
	if len(data) == 0 {
		return nil, nil // Игра не найдена
	}

	// Конвертируем данные Redis в структуру Game
	game, err := models.NewGameFromRedis(data)
	if err != nil {
		gs.logger.Errorf("Ошибка при парсинге данных игры %s:%s: %v", clubID, roomID, err)
		return nil, err
	}

	// Заполняем дополнительные поля
	game.ClubID = clubID
	game.RoomID = roomID

	return game, nil
}

// GetRoomInfo - получает информацию о комнате из Redis
func (gs *GameStateService) GetRoomInfo(clubID, roomID string) (*models.Room, error) {
	// Получаем ключ для информации о комнате
	roomKey := gs.redis.GetKeys().RoomInfo(clubID, roomID)

	// Читаем все поля hash
	data, err := gs.redis.HGetAll(roomKey)
	if err != nil {
		gs.logger.Errorf("Ошибка при получении информации о комнате %s:%s: %v", clubID, roomID, err)
		return nil, err
	}

	// Проверяем, что данные существуют
	if len(data) == 0 {
		return nil, nil // Комната не найдена
	}

	// Конвертируем данные Redis в структуру Room
	room, err := models.NewRoomFromRedis(data)
	if err != nil {
		gs.logger.Errorf("Ошибка при парсинге данных комнаты %s:%s: %v", clubID, roomID, err)
		return nil, err
	}

	return room, nil
}

// GetPlayersCount - получает количество игроков в комнате
func (gs *GameStateService) GetPlayersCount(clubID, roomID string) (int64, error) {
	playersKey := gs.redis.GetKeys().RoomPlayers(clubID, roomID)
	count, err := gs.redis.SCard(playersKey)
	if err != nil {
		gs.logger.Errorf("Ошибка при получении количества игроков в комнате %s:%s: %v", clubID, roomID, err)
		return 0, err
	}
	return count, nil
}

// GetSpectatorsCount - получает количество наблюдателей в комнате
func (gs *GameStateService) GetSpectatorsCount(clubID, roomID string) (int64, error) {
	spectatorsKey := gs.redis.GetKeys().RoomSpectators(clubID, roomID)
	count, err := gs.redis.SCard(spectatorsKey)
	if err != nil {
		gs.logger.Errorf("Ошибка при получении количества наблюдателей в комнате %s:%s: %v", clubID, roomID, err)
		return 0, err
	}
	return count, nil
}

// GetPlayerIDs - получает список ID всех игроков в комнате
func (gs *GameStateService) GetPlayerIDs(clubID, roomID string) ([]string, error) {
	playersKey := gs.redis.GetKeys().RoomPlayers(clubID, roomID)
	playerIDs, err := gs.redis.SMembers(playersKey)
	if err != nil {
		gs.logger.Errorf("Ошибка при получении списка игроков в комнате %s:%s: %v", clubID, roomID, err)
		return nil, err
	}
	return playerIDs, nil
}

// GetPlayer - получает информацию об игроке
func (gs *GameStateService) GetPlayer(clubID, roomID, userID string) (*models.Player, error) {
	playerKey := gs.redis.GetKeys().PlayerInfo(clubID, roomID, userID)
	data, err := gs.redis.HGetAll(playerKey)
	if err != nil {
		gs.logger.Errorf("Ошибка при получении данных игрока %s в комнате %s:%s: %v", userID, clubID, roomID, err)
		return nil, err
	}

	if len(data) == 0 {
		return nil, nil // Игрок не найден
	}

	player, err := models.NewPlayerFromRedis(data)
	if err != nil {
		gs.logger.Errorf("Ошибка при парсинге данных игрока %s: %v", userID, err)
		return nil, err
	}

	return player, nil
}

// === МЕТОДЫ ДЛЯ ОБНОВЛЕНИЯ СОСТОЯНИЯ ===

// UpdateGamePhase - обновляет фазу игры
func (gs *GameStateService) UpdateGamePhase(clubID, roomID string, phase models.GamePhase) error {
	gameKey := gs.redis.GetKeys().GameState(clubID, roomID)
	err := gs.redis.HSet(gameKey, "phase", string(phase))
	if err != nil {
		gs.logger.Errorf("Ошибка при обновлении фазы игры %s:%s: %v", clubID, roomID, err)
		return err
	}
	return nil
}

// UpdateRoomStatus - обновляет статус комнаты
func (gs *GameStateService) UpdateRoomStatus(clubID, roomID string, status models.RoomStatus) error {
	roomKey := gs.redis.GetKeys().RoomInfo(clubID, roomID)
	err := gs.redis.HSet(roomKey, "status", string(status))
	if err != nil {
		gs.logger.Errorf("Ошибка при обновлении статуса комнаты %s:%s: %v", clubID, roomID, err)
		return err
	}
	return nil
}

// UpdateGameState - обновляет несколько полей состояния игры одновременно
func (gs *GameStateService) UpdateGameState(clubID, roomID string, updates map[string]interface{}) error {
	gameKey := gs.redis.GetKeys().GameState(clubID, roomID)
	err := gs.redis.HMSet(gameKey, updates)
	if err != nil {
		gs.logger.Errorf("Ошибка при обновлении состояния игры %s:%s: %v", clubID, roomID, err)
		return err
	}
	return nil
}

// UpdateRoomInfo - обновляет несколько полей информации о комнате
func (gs *GameStateService) UpdateRoomInfo(clubID, roomID string, updates map[string]interface{}) error {
	roomKey := gs.redis.GetKeys().RoomInfo(clubID, roomID)
	err := gs.redis.HMSet(roomKey, updates)
	if err != nil {
		gs.logger.Errorf("Ошибка при обновлении информации о комнате %s:%s: %v", clubID, roomID, err)
		return err
	}
	return nil
}

// === ПРОВЕРКИ СОСТОЯНИЯ ===

// IsGameActive - проверяет, активна ли игра в комнате
func (gs *GameStateService) IsGameActive(clubID, roomID string) (bool, error) {
	game, err := gs.GetGameState(clubID, roomID)
	if err != nil {
		return false, err
	}
	if game == nil {
		return false, nil
	}
	return game.IsActive(), nil
}

// IsGameWaiting - проверяет, ожидает ли игра начала
func (gs *GameStateService) IsGameWaiting(clubID, roomID string) (bool, error) {
	game, err := gs.GetGameState(clubID, roomID)
	if err != nil {
		return false, err
	}
	if game == nil {
		return true, nil // Если игры нет, то она "ожидает"
	}
	return game.IsWaiting(), nil
}

// CanStartGame - проверяет, можно ли начать игру в комнате
func (gs *GameStateService) CanStartGame(clubID, roomID string, minPlayers int) (bool, error) {
	// Проверяем количество игроков
	playersCount, err := gs.GetPlayersCount(clubID, roomID)
	if err != nil {
		return false, err
	}

	if playersCount < int64(minPlayers) {
		return false, nil
	}

	// Проверяем, что игра не активна
	isActive, err := gs.IsGameActive(clubID, roomID)
	if err != nil {
		return false, err
	}

	return !isActive, nil
}

// HasMinimumPlayers - проверяет, достаточно ли игроков для продолжения игры
func (gs *GameStateService) HasMinimumPlayers(clubID, roomID string, minPlayers int) (bool, error) {
	playersCount, err := gs.GetPlayersCount(clubID, roomID)
	if err != nil {
		return false, err
	}
	return playersCount >= int64(minPlayers), nil
}

// RoomExists - проверяет, существует ли комната в Redis
func (gs *GameStateService) RoomExists(clubID, roomID string) (bool, error) {
	roomKey := gs.redis.GetKeys().RoomInfo(clubID, roomID)
	exists, err := gs.redis.Exists(roomKey)
	if err != nil {
		gs.logger.Errorf("Ошибка при проверке существования комнаты %s:%s: %v", clubID, roomID, err)
		return false, err
	}
	return exists, nil
}

// === МЕТОДЫ ДЛЯ РАБОТЫ С ИГРОКАМИ ===

// IsPlayerInRoom - проверяет, находится ли игрок в комнате (за столом)
func (gs *GameStateService) IsPlayerInRoom(clubID, roomID, userID string) (bool, error) {
	playersKey := gs.redis.GetKeys().RoomPlayers(clubID, roomID)
	isMember, err := gs.redis.SIsMember(playersKey, userID)
	if err != nil {
		gs.logger.Errorf("Ошибка при проверке присутствия игрока %s в комнате %s:%s: %v", userID, clubID, roomID, err)
		return false, err
	}
	return isMember, nil
}

// IsSpectatorInRoom - проверяет, является ли пользователь наблюдателем
func (gs *GameStateService) IsSpectatorInRoom(clubID, roomID, userID string) (bool, error) {
	spectatorsKey := gs.redis.GetKeys().RoomSpectators(clubID, roomID)
	isMember, err := gs.redis.SIsMember(spectatorsKey, userID)
	if err != nil {
		gs.logger.Errorf("Ошибка при проверке наблюдателя %s в комнате %s:%s: %v", userID, clubID, roomID, err)
		return false, err
	}
	return isMember, nil
}

// GetActivePlayers - получает всех активных игроков (не folded, не sit_out)
func (gs *GameStateService) GetActivePlayers(clubID, roomID string) ([]*models.Player, error) {
	// Получаем список всех игроков
	playerIDs, err := gs.GetPlayerIDs(clubID, roomID)
	if err != nil {
		return nil, err
	}

	// Получаем информацию о каждом игроке и фильтруем активных
	activePlayers := make([]*models.Player, 0)
	for _, userID := range playerIDs {
		player, err := gs.GetPlayer(clubID, roomID, userID)
		if err != nil {
			gs.logger.Warningf("Ошибка при получении данных игрока %s: %v", userID, err)
			continue
		}
		if player != nil && player.IsActive() {
			activePlayers = append(activePlayers, player)
		}
	}

	return activePlayers, nil
}

// === ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ ===

// GetFullRoomState - получает полное состояние комнаты (room + game + players)
// Удобно для отправки клиентам
func (gs *GameStateService) GetFullRoomState(clubID, roomID string) (map[string]interface{}, error) {
	// Получаем информацию о комнате
	room, err := gs.GetRoomInfo(clubID, roomID)
	if err != nil {
		return nil, err
	}

	// Получаем состояние игры
	game, err := gs.GetGameState(clubID, roomID)
	if err != nil {
		return nil, err
	}

	// Получаем количество игроков и наблюдателей
	playersCount, _ := gs.GetPlayersCount(clubID, roomID)
	spectatorsCount, _ := gs.GetSpectatorsCount(clubID, roomID)

	// Формируем полное состояние
	fullState := map[string]interface{}{
		"room":             room,
		"game":             game,
		"players_count":    playersCount,
		"spectators_count": spectatorsCount,
	}

	return fullState, nil
}

// CleanupRoom - очищает все данные комнаты из Redis
// ВНИМАНИЕ: Используйте с осторожностью! Это удалит ВСЕ данные комнаты.
func (gs *GameStateService) CleanupRoom(clubID, roomID string) error {
	keys := gs.redis.GetKeys()

	// Список всех ключей, которые нужно удалить
	keysToDelete := []string{
		keys.RoomInfo(clubID, roomID),
		keys.GameState(clubID, roomID),
		keys.RoomPlayers(clubID, roomID),
		keys.RoomSpectators(clubID, roomID),
		keys.RoomActions(clubID, roomID),
		keys.RoomTurnOrder(clubID, roomID),
		keys.RoomOccupiedSeats(clubID, roomID),
		keys.RoomDeck(clubID, roomID),
		keys.RoomPots(clubID, roomID),
		keys.RoomTimers(clubID, roomID),
	}

	// Удаляем все ключи
	err := gs.redis.Del(keysToDelete...)
	if err != nil {
		gs.logger.Errorf("Ошибка при очистке данных комнаты %s:%s: %v", clubID, roomID, err)
		return err
	}

	gs.logger.Infof("Данные комнаты %s:%s очищены", clubID, roomID)
	return nil
}
