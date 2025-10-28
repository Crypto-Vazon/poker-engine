package storage

import "fmt"

// Keys - структура для генерации Redis ключей
// Централизованное место для всех ключей Redis
// Это предотвращает опечатки и упрощает изменение структуры ключей
type Keys struct{}

// NewKeys - создает новый генератор ключей
func NewKeys() *Keys {
	return &Keys{}
}

// === КЛЮЧИ КЛУБОВ ===

// ClubRoomsActive - возвращает ключ для sorted set активных комнат клуба
// Формат: "club:{clubId}:rooms:active"
// Пример: "club:1:rooms:active"
// Тип: ZSET (sorted set) - хранит roomId со временем создания как score
func (k *Keys) ClubRoomsActive(clubID string) string {
	return fmt.Sprintf("club:%s:rooms:active", clubID)
}

// ClubRoomsActivePattern - возвращает паттерн для поиска всех активных комнат всех клубов
// Формат: "club:*:rooms:active"
// Используется для сканирования всех клубов
func (k *Keys) ClubRoomsActivePattern() string {
	return "club:*:rooms:active"
}

// === КЛЮЧИ КОМНАТ ===

// RoomInfo - возвращает ключ для информации о комнате
// Формат: "club:{clubId}:room:{roomId}:info"
// Пример: "club:1:room:3:info"
// Тип: HASH - хранит room_id, club_id, key, max_players, status и т.д.
func (k *Keys) RoomInfo(clubID, roomID string) string {
	return fmt.Sprintf("club:%s:room:%s:info", clubID, roomID)
}

// GameState - возвращает ключ для состояния игры в комнате
// Формат: "club:{clubId}:room:{roomId}:game"
// Пример: "club:1:room:3:game"
// Тип: HASH - хранит game_id, phase, pot, current_bet и т.д.
func (k *Keys) GameState(clubID, roomID string) string {
	return fmt.Sprintf("club:%s:room:%s:game", clubID, roomID)
}

// RoomPlayers - возвращает ключ для множества игроков в комнате
// Формат: "club:{clubId}:room:{roomId}:players"
// Пример: "club:1:room:3:players"
// Тип: SET - хранит userId игроков, сидящих за столом
func (k *Keys) RoomPlayers(clubID, roomID string) string {
	return fmt.Sprintf("club:%s:room:%s:players", clubID, roomID)
}

// RoomSpectators - возвращает ключ для множества наблюдателей в комнате
// Формат: "club:{clubId}:room:{roomId}:spectators"
// Пример: "club:1:room:3:spectators"
// Тип: SET - хранит userId наблюдателей
func (k *Keys) RoomSpectators(clubID, roomID string) string {
	return fmt.Sprintf("club:%s:room:%s:spectators", clubID, roomID)
}

// RoomActions - возвращает ключ для истории действий в комнате
// Формат: "club:{clubId}:room:{roomId}:actions"
// Пример: "club:1:room:3:actions"
// Тип: LIST - хранит JSON объекты с действиями (join, leave, bet и т.д.)
func (k *Keys) RoomActions(clubID, roomID string) string {
	return fmt.Sprintf("club:%s:room:%s:actions", clubID, roomID)
}

// RoomTurnOrder - возвращает ключ для очереди ходов игроков
// Формат: "club:{clubId}:room:{roomId}:turn_order"
// Пример: "club:1:room:3:turn_order"
// Тип: ZSET - хранит userId с позицией за столом как score
func (k *Keys) RoomTurnOrder(clubID, roomID string) string {
	return fmt.Sprintf("club:%s:room:%s:turn_order", clubID, roomID)
}

// RoomOccupiedSeats - возвращает ключ для занятых мест за столом
// Формат: "club:{clubId}:room:{roomId}:occupied_seats"
// Пример: "club:1:room:3:occupied_seats"
// Тип: SET - хранит номера занятых мест (0, 1, 2, ... max_players-1)
func (k *Keys) RoomOccupiedSeats(clubID, roomID string) string {
	return fmt.Sprintf("club:%s:room:%s:occupied_seats", clubID, roomID)
}

// RoomDeck - возвращает ключ для колоды карт в комнате
// Формат: "club:{clubId}:room:{roomId}:deck"
// Пример: "club:1:room:3:deck"
// Тип: LIST - хранит оставшиеся карты в колоде
func (k *Keys) RoomDeck(clubID, roomID string) string {
	return fmt.Sprintf("club:%s:room:%s:deck", clubID, roomID)
}

// RoomPots - возвращает ключ для информации о банках
// Формат: "club:{clubId}:room:{roomId}:pots"
// Пример: "club:1:room:3:pots"
// Тип: HASH - хранит main_pot и side_pots (JSON массив)
func (k *Keys) RoomPots(clubID, roomID string) string {
	return fmt.Sprintf("club:%s:room:%s:pots", clubID, roomID)
}

// RoomTimers - возвращает ключ для таймеров комнаты
// Формат: "club:{clubId}:room:{roomId}:timers"
// Пример: "club:1:room:3:timers"
// Тип: HASH - хранит turn_start_time, turn_duration, time_bank
func (k *Keys) RoomTimers(clubID, roomID string) string {
	return fmt.Sprintf("club:%s:room:%s:timers", clubID, roomID)
}

// === КЛЮЧИ ИГРОКОВ ===

// PlayerInfo - возвращает ключ для информации об игроке в комнате
// Формат: "club:{clubId}:room:{roomId}:player:{userId}"
// Пример: "club:1:room:3:player:456"
// Тип: HASH - хранит user_id, username, position, chips, bet, cards, status и т.д.
func (k *Keys) PlayerInfo(clubID, roomID, userID string) string {
	return fmt.Sprintf("club:%s:room:%s:player:%s", clubID, roomID, userID)
}

// SpectatorInfo - возвращает ключ для информации о наблюдателе
// Формат: "club:{clubId}:room:{roomId}:spectator:{userId}"
// Пример: "club:1:room:3:spectator:456"
// Тип: HASH - хранит user_id, username, joined_at
func (k *Keys) SpectatorInfo(clubID, roomID, userID string) string {
	return fmt.Sprintf("club:%s:room:%s:spectator:%s", clubID, roomID, userID)
}

// UserCurrentRoom - возвращает ключ для обратного индекса (в какой комнате пользователь)
// Формат: "club:{clubId}:user:{userId}:current_room"
// Пример: "club:1:user:456:current_room"
// Тип: STRING - хранит roomId, где находится пользователь
// Используется для быстрой проверки O(1)
func (k *Keys) UserCurrentRoom(clubID, userID string) string {
	return fmt.Sprintf("club:%s:user:%s:current_room", clubID, userID)
}

// === ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ ===

// IsRoomKey - проверяет, является ли ключ ключом комнаты
// Возвращает true если ключ начинается с "club:{clubId}:room:"
func (k *Keys) IsRoomKey(key string) bool {
	return len(key) > 5 && key[:5] == "club:"
}

// ExtractClubID - извлекает clubId из ключа
// Пример: "club:1:room:3:info" -> "1"
// Возвращает пустую строку если формат некорректный
func (k *Keys) ExtractClubID(key string) string {
	// Формат: "club:{clubId}:..."
	// Ищем первое и второе двоеточие
	start := 5 // длина "club:"
	if len(key) <= start {
		return ""
	}

	end := start
	for end < len(key) && key[end] != ':' {
		end++
	}

	if end == start || end >= len(key) {
		return ""
	}

	return key[start:end]
}

// ExtractRoomID - извлекает roomId из ключа
// Пример: "club:1:room:3:info" -> "3"
// Возвращает пустую строку если формат некорректный
func (k *Keys) ExtractRoomID(key string) string {
	// Формат: "club:{clubId}:room:{roomId}:..."
	// Ищем "room:" и следующее двоеточие
	roomPrefix := ":room:"
	roomIdx := indexOf(key, roomPrefix)
	if roomIdx == -1 {
		return ""
	}

	start := roomIdx + len(roomPrefix)
	if start >= len(key) {
		return ""
	}

	end := start
	for end < len(key) && key[end] != ':' {
		end++
	}

	if end == start {
		return ""
	}

	return key[start:end]
}

// ExtractUserID - извлекает userId из ключа игрока/наблюдателя
// Пример: "club:1:room:3:player:456" -> "456"
// Возвращает пустую строку если формат некорректный
func (k *Keys) ExtractUserID(key string) string {
	// Формат: "...player:{userId}" или "...spectator:{userId}"
	// Ищем последнее двоеточие
	lastColon := lastIndexOf(key, ":")
	if lastColon == -1 || lastColon == len(key)-1 {
		return ""
	}

	return key[lastColon+1:]
}

// === СЛУЖЕБНЫЕ ФУНКЦИИ ===

// indexOf - находит индекс первого вхождения подстроки
// Возвращает -1 если подстрока не найдена
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// lastIndexOf - находит индекс последнего вхождения символа
// Возвращает -1 если символ не найден
func lastIndexOf(s, char string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if string(s[i]) == char {
			return i
		}
	}
	return -1
}

// === ГЛОБАЛЬНЫЙ ЭКЗЕМПЛЯР ===

// DefaultKeys - глобальный экземпляр генератора ключей для удобства
var DefaultKeys = NewKeys()

// Глобальные функции для быстрого доступа без создания экземпляра

func ClubRoomsActive(clubID string) string {
	return DefaultKeys.ClubRoomsActive(clubID)
}

func ClubRoomsActivePattern() string {
	return DefaultKeys.ClubRoomsActivePattern()
}

func RoomInfo(clubID, roomID string) string {
	return DefaultKeys.RoomInfo(clubID, roomID)
}

func GameState(clubID, roomID string) string {
	return DefaultKeys.GameState(clubID, roomID)
}

func RoomPlayers(clubID, roomID string) string {
	return DefaultKeys.RoomPlayers(clubID, roomID)
}

func RoomSpectators(clubID, roomID string) string {
	return DefaultKeys.RoomSpectators(clubID, roomID)
}

func RoomActions(clubID, roomID string) string {
	return DefaultKeys.RoomActions(clubID, roomID)
}

func PlayerInfo(clubID, roomID, userID string) string {
	return DefaultKeys.PlayerInfo(clubID, roomID, userID)
}

func SpectatorInfo(clubID, roomID, userID string) string {
	return DefaultKeys.SpectatorInfo(clubID, roomID, userID)
}

func UserCurrentRoom(clubID, userID string) string {
	return DefaultKeys.UserCurrentRoom(clubID, userID)
}
