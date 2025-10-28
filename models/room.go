package models

import (
	"strconv"
)

// RoomStatus - тип для статуса комнаты
type RoomStatus string

// Константы статусов комнаты
const (
	// RoomStatusWaiting - комната ожидает игроков (меньше минимума)
	RoomStatusWaiting RoomStatus = "waiting"

	// RoomStatusReady - комната готова к игре (достаточно игроков, но игра не началась)
	RoomStatusReady RoomStatus = "ready"

	// RoomStatusGaming - в комнате идет игра
	RoomStatusGaming RoomStatus = "gaming"

	// RoomStatusPaused - игра приостановлена (например, при отключении игрока)
	RoomStatusPaused RoomStatus = "paused"

	// RoomStatusClosed - комната закрыта (больше не принимает игроков)
	RoomStatusClosed RoomStatus = "closed"
)

// Room - основная структура комнаты (полная информация)
type Room struct {
	// ID комнаты
	RoomID string

	// ID клуба, которому принадлежит комната
	ClubID string

	// Ключ комнаты (уникальный идентификатор)
	Key string

	// Максимальное количество игроков за столом (обычно 2-9)
	MaxPlayers int

	// Малый блайнд
	SmallBlind int

	// Большой блайнд
	BigBlind int

	// Минимальный buy-in (входная ставка)
	BuyInMin int

	// Максимальный buy-in
	BuyInMax int

	// Валюта (например: "chips", "USD", "EUR")
	Currency string

	// Текущий статус комнаты
	Status RoomStatus

	// Время создания комнаты (ISO 8601)
	CreatedAt string

	// Количество текущих игроков за столом
	CurrentPlayers int

	// Количество наблюдателей
	CurrentSpectators int
}

// RoomInfo - упрощенная структура для информации о комнате из Redis
// Соответствует hash "club:{clubId}:room:{roomId}:info"
type RoomInfo struct {
	RoomID     string     `json:"room_id"`
	ClubID     string     `json:"club_id"`
	Key        string     `json:"key"`
	MaxPlayers string     `json:"max_players"`
	SmallBlind string     `json:"small_blind"`
	BigBlind   string     `json:"big_blind"`
	BuyInMin   string     `json:"buy_in_min"`
	BuyInMax   string     `json:"buy_in_max"`
	Currency   string     `json:"currency"`
	Status     RoomStatus `json:"status"`
	CreatedAt  string     `json:"created_at"`
}

// NewRoomFromRedis - создает Room из данных Redis hash
func NewRoomFromRedis(data map[string]string) (*Room, error) {
	// Парсим числовые поля (они хранятся как строки в Redis)
	maxPlayers, _ := strconv.Atoi(data["max_players"])
	smallBlind, _ := strconv.Atoi(data["small_blind"])
	bigBlind, _ := strconv.Atoi(data["big_blind"])
	buyInMin, _ := strconv.Atoi(data["buy_in_min"])
	buyInMax, _ := strconv.Atoi(data["buy_in_max"])

	return &Room{
		RoomID:     data["room_id"],
		ClubID:     data["club_id"],
		Key:        data["key"],
		MaxPlayers: maxPlayers,
		SmallBlind: smallBlind,
		BigBlind:   bigBlind,
		BuyInMin:   buyInMin,
		BuyInMax:   buyInMax,
		Currency:   data["currency"],
		Status:     RoomStatus(data["status"]),
		CreatedAt:  data["created_at"],
	}, nil
}

// ToRedisHash - преобразует Room в map для сохранения в Redis hash
func (r *Room) ToRedisHash() map[string]interface{} {
	return map[string]interface{}{
		"room_id":     r.RoomID,
		"club_id":     r.ClubID,
		"key":         r.Key,
		"max_players": r.MaxPlayers,
		"small_blind": r.SmallBlind,
		"big_blind":   r.BigBlind,
		"buy_in_min":  r.BuyInMin,
		"buy_in_max":  r.BuyInMax,
		"currency":    r.Currency,
		"status":      string(r.Status),
		"created_at":  r.CreatedAt,
	}
}

// === МЕТОДЫ ДЛЯ ВАЛИДАЦИИ ===

// IsStatusValid - проверяет, что статус корректный
func (r *Room) IsStatusValid() bool {
	switch r.Status {
	case RoomStatusWaiting, RoomStatusReady, RoomStatusGaming, RoomStatusPaused, RoomStatusClosed:
		return true
	default:
		return false
	}
}

// CanStartGame - проверяет, можно ли начать игру в комнате
// Условия: статус waiting/ready и достаточно игроков
func (r *Room) CanStartGame(minPlayers int) bool {
	return (r.Status == RoomStatusWaiting || r.Status == RoomStatusReady) &&
		r.CurrentPlayers >= minPlayers
}

// CanAcceptPlayers - проверяет, может ли комната принять новых игроков
func (r *Room) CanAcceptPlayers() bool {
	return r.Status != RoomStatusClosed && r.CurrentPlayers < r.MaxPlayers
}

// IsFull - проверяет, заполнена ли комната
func (r *Room) IsFull() bool {
	return r.CurrentPlayers >= r.MaxPlayers
}

// IsEmpty - проверяет, пустая ли комната (нет игроков)
func (r *Room) IsEmpty() bool {
	return r.CurrentPlayers == 0
}

// HasMinimumPlayers - проверяет, достаточно ли игроков для игры
func (r *Room) HasMinimumPlayers(minPlayers int) bool {
	return r.CurrentPlayers >= minPlayers
}

// === МЕТОДЫ ДЛЯ ИЗМЕНЕНИЯ СОСТОЯНИЯ ===

// SetStatus - устанавливает новый статус комнаты
func (r *Room) SetStatus(status RoomStatus) {
	r.Status = status
}

// IncrementPlayers - увеличивает счетчик игроков
func (r *Room) IncrementPlayers() {
	if r.CurrentPlayers < r.MaxPlayers {
		r.CurrentPlayers++
	}
}

// DecrementPlayers - уменьшает счетчик игроков
func (r *Room) DecrementPlayers() {
	if r.CurrentPlayers > 0 {
		r.CurrentPlayers--
	}
}

// IncrementSpectators - увеличивает счетчик наблюдателей
func (r *Room) IncrementSpectators() {
	r.CurrentSpectators++
}

// DecrementSpectators - уменьшает счетчик наблюдателей
func (r *Room) DecrementSpectators() {
	if r.CurrentSpectators > 0 {
		r.CurrentSpectators--
	}
}

// === ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ ===

// GetIdentifier - возвращает уникальный идентификатор комнаты
// Формат: "club_{clubId}_room_{roomId}"
func (r *Room) GetIdentifier() string {
	return "club_" + r.ClubID + "_room_" + r.RoomID
}

// GetBlindInfo - возвращает информацию о блайндах в читаемом виде
// Формат: "SB: 10 / BB: 20"
func (r *Room) GetBlindInfo() string {
	return "SB: " + strconv.Itoa(r.SmallBlind) + " / BB: " + strconv.Itoa(r.BigBlind)
}

// GetBuyInInfo - возвращает информацию о buy-in в читаемом виде
// Формат: "Min: 100 / Max: 1000"
func (r *Room) GetBuyInInfo() string {
	return "Min: " + strconv.Itoa(r.BuyInMin) + " / Max: " + strconv.Itoa(r.BuyInMax)
}

// GetAvailableSeats - возвращает количество свободных мест
func (r *Room) GetAvailableSeats() int {
	available := r.MaxPlayers - r.CurrentPlayers
	if available < 0 {
		return 0
	}
	return available
}

// GetOccupancyPercentage - возвращает процент заполненности комнаты
func (r *Room) GetOccupancyPercentage() int {
	if r.MaxPlayers == 0 {
		return 0
	}
	return (r.CurrentPlayers * 100) / r.MaxPlayers
}

// === СТАТИЧЕСКИЕ ФУНКЦИИ ===

// ParseRoomStatus - парсит строку в RoomStatus
func ParseRoomStatus(status string) RoomStatus {
	switch status {
	case "waiting":
		return RoomStatusWaiting
	case "ready":
		return RoomStatusReady
	case "gaming":
		return RoomStatusGaming
	case "paused":
		return RoomStatusPaused
	case "closed":
		return RoomStatusClosed
	default:
		return RoomStatusWaiting // По умолчанию
	}
}

// IsValidRoomStatus - проверяет, является ли строка валидным статусом
func IsValidRoomStatus(status string) bool {
	switch status {
	case "waiting", "ready", "gaming", "paused", "closed":
		return true
	default:
		return false
	}
}

// GetAllRoomStatuses - возвращает все возможные статусы комнаты
func GetAllRoomStatuses() []RoomStatus {
	return []RoomStatus{
		RoomStatusWaiting,
		RoomStatusReady,
		RoomStatusGaming,
		RoomStatusPaused,
		RoomStatusClosed,
	}
}

// === КОНСТАНТЫ ===

const (
	// Стандартные значения по умолчанию
	DefaultMaxPlayers = 9
	DefaultSmallBlind = 10
	DefaultBigBlind   = 20
	DefaultBuyInMin   = 100
	DefaultBuyInMax   = 1000
	DefaultCurrency   = "chips"

	// Минимальное количество игроков для начала игры
	MinPlayersToStart = 2

	// Максимально возможное количество игроков за столом
	AbsoluteMaxPlayers = 10
)
