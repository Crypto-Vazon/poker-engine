package models

import (
	"encoding/json"
	"strconv"
)

// PlayerStatus - тип для статуса игрока
type PlayerStatus string

// Константы статусов игрока
const (
	// PlayerStatusWaiting - игрок ожидает начала раздачи
	PlayerStatusWaiting PlayerStatus = "waiting"

	// PlayerStatusActive - игрок активен в текущей раздаче
	PlayerStatusActive PlayerStatus = "active"

	// PlayerStatusFolded - игрок сбросил карты
	PlayerStatusFolded PlayerStatus = "folded"

	// PlayerStatusAllIn - игрок поставил все фишки
	PlayerStatusAllIn PlayerStatus = "all_in"

	// PlayerStatusSitOut - игрок пропускает раздачи (сидит вне игры)
	PlayerStatusSitOut PlayerStatus = "sit_out"
)

// PlayerAction - тип для действия игрока
type PlayerAction string

// Константы действий игрока
const (
	ActionCheck PlayerAction = "check"  // Пас (когда нет ставки)
	ActionCall  PlayerAction = "call"   // Колл (уравнять ставку)
	ActionBet   PlayerAction = "bet"    // Ставка (когда нет текущей ставки)
	ActionRaise PlayerAction = "raise"  // Рейз (увеличить ставку)
	ActionFold  PlayerAction = "fold"   // Фолд (сбросить карты)
	ActionAllIn PlayerAction = "all_in" // Олл-ин (все фишки)
	ActionBlind PlayerAction = "blind"  // Блайнд (обязательная ставка)
)

// Player - полная структура игрока
type Player struct {
	// ID пользователя
	UserID string

	// Имя пользователя
	Username string

	// Позиция за столом (seat number: 0 - maxPlayers-1)
	Position int

	// Количество фишек у игрока
	Chips int

	// Текущая ставка игрока в раунде
	Bet int

	// Карты игрока (обычно 2 карты в техасском холдеме)
	Cards []string

	// Текущий статус игрока
	Status PlayerStatus

	// Последнее действие игрока
	LastAction *PlayerAction

	// Является ли игрок дилером в текущей раздаче
	IsDealer bool

	// Является ли игрок малым блайндом
	IsSmallBlind bool

	// Является ли игрок большим блайндом
	IsBigBlind bool

	// Время, когда игрок сел за стол (ISO 8601)
	JoinedTableAt string

	// Общая сумма фишек, с которыми игрок сел за стол (buy-in)
	InitialBuyIn int
}

// PlayerInfo - структура информации об игроке из Redis
// Соответствует hash "club:{clubId}:room:{roomId}:player:{userId}"
type PlayerInfo struct {
	UserID        string       `json:"user_id"`
	Username      string       `json:"username"`
	Position      string       `json:"position"`
	Chips         string       `json:"chips"`
	Bet           string       `json:"bet"`
	Cards         string       `json:"cards"` // JSON массив
	Status        PlayerStatus `json:"status"`
	LastAction    string       `json:"last_action"`
	IsDealer      string       `json:"is_dealer"`
	IsSmallBlind  string       `json:"is_small_blind"`
	IsBigBlind    string       `json:"is_big_blind"`
	JoinedTableAt string       `json:"joined_table_at"`
}

// NewPlayerFromRedis - создает Player из данных Redis hash
func NewPlayerFromRedis(data map[string]string) (*Player, error) {
	// Парсим числовые поля
	position, _ := strconv.Atoi(data["position"])
	chips, _ := strconv.Atoi(data["chips"])
	bet, _ := strconv.Atoi(data["bet"])

	// Парсим карты из JSON
	var cards []string
	if data["cards"] != "" {
		json.Unmarshal([]byte(data["cards"]), &cards)
	}

	// Парсим последнее действие
	var lastAction *PlayerAction
	if data["last_action"] != "" && data["last_action"] != "null" {
		action := PlayerAction(data["last_action"])
		lastAction = &action
	}

	// Парсим boolean поля (в Redis хранятся как строки)
	isDealer := data["is_dealer"] == "true" || data["is_dealer"] == "1"
	isSmallBlind := data["is_small_blind"] == "true" || data["is_small_blind"] == "1"
	isBigBlind := data["is_big_blind"] == "true" || data["is_big_blind"] == "1"

	return &Player{
		UserID:        data["user_id"],
		Username:      data["username"],
		Position:      position,
		Chips:         chips,
		Bet:           bet,
		Cards:         cards,
		Status:        PlayerStatus(data["status"]),
		LastAction:    lastAction,
		IsDealer:      isDealer,
		IsSmallBlind:  isSmallBlind,
		IsBigBlind:    isBigBlind,
		JoinedTableAt: data["joined_table_at"],
	}, nil
}

// ToRedisHash - преобразует Player в map для сохранения в Redis hash
func (p *Player) ToRedisHash() map[string]interface{} {
	hash := map[string]interface{}{
		"user_id":         p.UserID,
		"username":        p.Username,
		"position":        p.Position,
		"chips":           p.Chips,
		"bet":             p.Bet,
		"status":          string(p.Status),
		"is_dealer":       p.IsDealer,
		"is_small_blind":  p.IsSmallBlind,
		"is_big_blind":    p.IsBigBlind,
		"joined_table_at": p.JoinedTableAt,
	}

	// Cards как JSON
	cardsJSON, _ := json.Marshal(p.Cards)
	hash["cards"] = string(cardsJSON)

	// LastAction
	if p.LastAction != nil {
		hash["last_action"] = string(*p.LastAction)
	} else {
		hash["last_action"] = ""
	}

	return hash
}

// === МЕТОДЫ ДЛЯ РАБОТЫ С ФИШКАМИ ===

// AddChips - добавляет фишки игроку
func (p *Player) AddChips(amount int) {
	p.Chips += amount
}

// RemoveChips - удаляет фишки у игрока
// Возвращает false если недостаточно фишек
func (p *Player) RemoveChips(amount int) bool {
	if p.Chips < amount {
		return false
	}
	p.Chips -= amount
	return true
}

// HasChips - проверяет, есть ли у игрока фишки
func (p *Player) HasChips() bool {
	return p.Chips > 0
}

// GetTotalChips - возвращает общее количество фишек (стек + ставка)
func (p *Player) GetTotalChips() int {
	return p.Chips + p.Bet
}

// === МЕТОДЫ ДЛЯ РАБОТЫ СО СТАВКАМИ ===

// PlaceBet - делает ставку (перемещает фишки из стека в ставку)
// Возвращает false если недостаточно фишек
func (p *Player) PlaceBet(amount int) bool {
	if p.Chips < amount {
		return false
	}
	p.Chips -= amount
	p.Bet += amount
	return true
}

// ResetBet - сбрасывает ставку игрока (начало нового раунда торговли)
func (p *Player) ResetBet() {
	p.Bet = 0
}

// GoAllIn - игрок ставит все фишки
func (p *Player) GoAllIn() {
	p.Bet += p.Chips
	p.Chips = 0
	p.Status = PlayerStatusAllIn
}

// === МЕТОДЫ ДЛЯ РАБОТЫ СО СТАТУСОМ ===

// SetStatus - устанавливает новый статус игрока
func (p *Player) SetStatus(status PlayerStatus) {
	p.Status = status
}

// IsActive - проверяет, активен ли игрок в раздаче
func (p *Player) IsActive() bool {
	return p.Status == PlayerStatusActive
}

// IsFolded - проверяет, сбросил ли игрок карты
func (p *Player) IsFolded() bool {
	return p.Status == PlayerStatusFolded
}

// IsAllIn - проверяет, в олл-ине ли игрок
func (p *Player) IsAllIn() bool {
	return p.Status == PlayerStatusAllIn
}

// IsSittingOut - проверяет, сидит ли игрок вне игры
func (p *Player) IsSittingOut() bool {
	return p.Status == PlayerStatusSitOut
}

// CanAct - проверяет, может ли игрок совершать действия
func (p *Player) CanAct() bool {
	return p.Status == PlayerStatusActive && p.HasChips()
}

// === МЕТОДЫ ДЛЯ РАБОТЫ С КАРТАМИ ===

// SetCards - устанавливает карты игрока
func (p *Player) SetCards(cards []string) {
	p.Cards = cards
}

// AddCard - добавляет карту игроку
func (p *Player) AddCard(card string) {
	p.Cards = append(p.Cards, card)
}

// ClearCards - очищает карты игрока
func (p *Player) ClearCards() {
	p.Cards = []string{}
}

// HasCards - проверяет, есть ли у игрока карты
func (p *Player) HasCards() bool {
	return len(p.Cards) > 0
}

// GetCardsCount - возвращает количество карт у игрока
func (p *Player) GetCardsCount() int {
	return len(p.Cards)
}

// === МЕТОДЫ ДЛЯ РАБОТЫ С ДЕЙСТВИЯМИ ===

// SetLastAction - устанавливает последнее действие игрока
func (p *Player) SetLastAction(action PlayerAction) {
	p.LastAction = &action
}

// ClearLastAction - очищает последнее действие
func (p *Player) ClearLastAction() {
	p.LastAction = nil
}

// GetLastActionString - возвращает последнее действие как строку
func (p *Player) GetLastActionString() string {
	if p.LastAction == nil {
		return ""
	}
	return string(*p.LastAction)
}

// === МЕТОДЫ ДЛЯ РАБОТЫ С ПОЗИЦИЯМИ ===

// SetDealer - устанавливает/снимает флаг дилера
func (p *Player) SetDealer(isDealer bool) {
	p.IsDealer = isDealer
}

// SetSmallBlind - устанавливает/снимает флаг малого блайнда
func (p *Player) SetSmallBlind(isSmallBlind bool) {
	p.IsSmallBlind = isSmallBlind
}

// SetBigBlind - устанавливает/снимает флаг большого блайнда
func (p *Player) SetBigBlind(isBigBlind bool) {
	p.IsBigBlind = isBigBlind
}

// ClearPositionFlags - очищает все флаги позиций
func (p *Player) ClearPositionFlags() {
	p.IsDealer = false
	p.IsSmallBlind = false
	p.IsBigBlind = false
}

// === ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ ===

// GetIdentifier - возвращает уникальный идентификатор игрока
func (p *Player) GetIdentifier() string {
	return "user_" + p.UserID
}

// GetDisplayName - возвращает имя для отображения
func (p *Player) GetDisplayName() string {
	if p.Username != "" {
		return p.Username
	}
	return "User_" + p.UserID
}

// GetPositionString - возвращает позицию как строку
func (p *Player) GetPositionString() string {
	return "Seat " + strconv.Itoa(p.Position)
}

// === СТАТИЧЕСКИЕ ФУНКЦИИ ===

// ParsePlayerStatus - парсит строку в PlayerStatus
func ParsePlayerStatus(status string) PlayerStatus {
	switch status {
	case "waiting":
		return PlayerStatusWaiting
	case "active":
		return PlayerStatusActive
	case "folded":
		return PlayerStatusFolded
	case "all_in":
		return PlayerStatusAllIn
	case "sit_out":
		return PlayerStatusSitOut
	default:
		return PlayerStatusWaiting
	}
}

// ParsePlayerAction - парсит строку в PlayerAction
func ParsePlayerAction(action string) PlayerAction {
	switch action {
	case "check":
		return ActionCheck
	case "call":
		return ActionCall
	case "bet":
		return ActionBet
	case "raise":
		return ActionRaise
	case "fold":
		return ActionFold
	case "all_in":
		return ActionAllIn
	case "blind":
		return ActionBlind
	default:
		return ActionFold
	}
}

// IsValidPlayerStatus - проверяет, является ли строка валидным статусом
func IsValidPlayerStatus(status string) bool {
	switch status {
	case "waiting", "active", "folded", "all_in", "sit_out":
		return true
	default:
		return false
	}
}

// IsValidPlayerAction - проверяет, является ли строка валидным действием
func IsValidPlayerAction(action string) bool {
	switch action {
	case "check", "call", "bet", "raise", "fold", "all_in", "blind":
		return true
	default:
		return false
	}
}

// GetAllPlayerStatuses - возвращает все возможные статусы игрока
func GetAllPlayerStatuses() []PlayerStatus {
	return []PlayerStatus{
		PlayerStatusWaiting,
		PlayerStatusActive,
		PlayerStatusFolded,
		PlayerStatusAllIn,
		PlayerStatusSitOut,
	}
}

// GetAllPlayerActions - возвращает все возможные действия игрока
func GetAllPlayerActions() []PlayerAction {
	return []PlayerAction{
		ActionCheck,
		ActionCall,
		ActionBet,
		ActionRaise,
		ActionFold,
		ActionAllIn,
		ActionBlind,
	}
}
