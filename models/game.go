package models

import (
	"encoding/json"
	"strconv"
	"time"
)

// GamePhase - тип для фазы игры
type GamePhase string

// Константы фаз покерной игры
const (
	// GamePhaseWaiting - ожидание начала игры (недостаточно игроков)
	GamePhaseWaiting GamePhase = "waiting"

	// GamePhasePreFlop - фаза до раздачи общих карт (только карты игроков)
	GamePhasePreFlop GamePhase = "pre_flop"

	// GamePhaseFlop - фаза после раздачи трех общих карт
	GamePhaseFlop GamePhase = "flop"

	// GamePhaseTurn - фаза после раздачи четвертой общей карты
	GamePhaseTurn GamePhase = "turn"

	// GamePhaseRiver - фаза после раздачи пятой общей карты
	GamePhaseRiver GamePhase = "river"

	// GamePhaseShowdown - вскрытие карт и определение победителя
	GamePhaseShowdown GamePhase = "showdown"

	// GamePhaseFinished - игра завершена (раздача окончена)
	GamePhaseFinished GamePhase = "finished"
)

// Game - полная структура игры
type Game struct {
	// Уникальный ID игры
	GameID string

	// ID комнаты, в которой идет игра
	RoomID string

	// ID клуба
	ClubID string

	// Текущая фаза игры
	Phase GamePhase

	// Общий банк (pot) - сумма всех ставок
	Pot int

	// Текущая ставка в раунде (для call)
	CurrentBet int

	// Позиция дилера (seat number)
	DealerPosition int

	// Позиция малого блайнда
	SmallBlindPosition *int

	// Позиция большого блайнда
	BigBlindPosition *int

	// Позиция текущего игрока (чей ход)
	CurrentPlayerPosition *int

	// Номер раунда (hand number)
	RoundNumber int

	// Общие карты на столе (community cards)
	CommunityCards []string

	// Время начала игры
	StartedAt *time.Time

	// Боковые банки (side pots) для all-in ситуаций
	SidePots []SidePot
}

// GameState - структура состояния игры из Redis
// Соответствует hash "club:{clubId}:room:{roomId}:game"
type GameState struct {
	GameID                string    `json:"game_id"`
	Phase                 GamePhase `json:"phase"`
	Pot                   string    `json:"pot"`
	CurrentBet            string    `json:"current_bet"`
	DealerPosition        string    `json:"dealer_position"`
	SmallBlindPosition    string    `json:"small_blind_position"`
	BigBlindPosition      string    `json:"big_blind_position"`
	CurrentPlayerPosition string    `json:"current_player_position"`
	RoundNumber           string    `json:"round_number"`
	CommunityCards        string    `json:"community_cards"` // JSON массив
	StartedAt             string    `json:"started_at"`      // ISO 8601
}

// SidePot - структура для бокового банка (когда игрок идет all-in)
type SidePot struct {
	// Сумма бокового банка
	Amount int `json:"amount"`

	// ID игроков, участвующих в этом банке
	EligiblePlayers []string `json:"eligible_players"`
}

// NewGameFromRedis - создает Game из данных Redis hash
func NewGameFromRedis(data map[string]string) (*Game, error) {
	// Парсим числовые поля
	pot, _ := strconv.Atoi(data["pot"])
	currentBet, _ := strconv.Atoi(data["current_bet"])
	dealerPosition, _ := strconv.Atoi(data["dealer_position"])
	roundNumber, _ := strconv.Atoi(data["round_number"])

	// Парсим nullable позиции
	var smallBlindPos, bigBlindPos, currentPlayerPos *int

	if data["small_blind_position"] != "" && data["small_blind_position"] != "null" {
		pos, _ := strconv.Atoi(data["small_blind_position"])
		smallBlindPos = &pos
	}

	if data["big_blind_position"] != "" && data["big_blind_position"] != "null" {
		pos, _ := strconv.Atoi(data["big_blind_position"])
		bigBlindPos = &pos
	}

	if data["current_player_position"] != "" && data["current_player_position"] != "null" {
		pos, _ := strconv.Atoi(data["current_player_position"])
		currentPlayerPos = &pos
	}

	// Парсим community cards из JSON
	var communityCards []string
	if data["community_cards"] != "" {
		json.Unmarshal([]byte(data["community_cards"]), &communityCards)
	}

	// Парсим started_at
	var startedAt *time.Time
	if data["started_at"] != "" && data["started_at"] != "null" {
		t, err := time.Parse(time.RFC3339, data["started_at"])
		if err == nil {
			startedAt = &t
		}
	}

	return &Game{
		GameID:                data["game_id"],
		Phase:                 GamePhase(data["phase"]),
		Pot:                   pot,
		CurrentBet:            currentBet,
		DealerPosition:        dealerPosition,
		SmallBlindPosition:    smallBlindPos,
		BigBlindPosition:      bigBlindPos,
		CurrentPlayerPosition: currentPlayerPos,
		RoundNumber:           roundNumber,
		CommunityCards:        communityCards,
		StartedAt:             startedAt,
	}, nil
}

// ToRedisHash - преобразует Game в map для сохранения в Redis hash
func (g *Game) ToRedisHash() map[string]interface{} {
	hash := map[string]interface{}{
		"game_id":         g.GameID,
		"phase":           string(g.Phase),
		"pot":             g.Pot,
		"current_bet":     g.CurrentBet,
		"dealer_position": g.DealerPosition,
		"round_number":    g.RoundNumber,
	}

	// Добавляем nullable поля
	if g.SmallBlindPosition != nil {
		hash["small_blind_position"] = *g.SmallBlindPosition
	} else {
		hash["small_blind_position"] = ""
	}

	if g.BigBlindPosition != nil {
		hash["big_blind_position"] = *g.BigBlindPosition
	} else {
		hash["big_blind_position"] = ""
	}

	if g.CurrentPlayerPosition != nil {
		hash["current_player_position"] = *g.CurrentPlayerPosition
	} else {
		hash["current_player_position"] = ""
	}

	// Community cards как JSON
	cardsJSON, _ := json.Marshal(g.CommunityCards)
	hash["community_cards"] = string(cardsJSON)

	// Started at
	if g.StartedAt != nil {
		hash["started_at"] = g.StartedAt.Format(time.RFC3339)
	} else {
		hash["started_at"] = ""
	}

	return hash
}

// === МЕТОДЫ ДЛЯ РАБОТЫ С ФАЗАМИ ===

// IsActive - проверяет, активна ли игра (идет раздача)
func (g *Game) IsActive() bool {
	return g.Phase != GamePhaseWaiting && g.Phase != GamePhaseFinished
}

// IsWaiting - проверяет, ожидает ли игра начала
func (g *Game) IsWaiting() bool {
	return g.Phase == GamePhaseWaiting
}

// IsFinished - проверяет, завершена ли игра
func (g *Game) IsFinished() bool {
	return g.Phase == GamePhaseFinished
}

// CanTransitionTo - проверяет, можно ли перейти к указанной фазе
func (g *Game) CanTransitionTo(nextPhase GamePhase) bool {
	// Определяем допустимые переходы между фазами
	validTransitions := map[GamePhase][]GamePhase{
		GamePhaseWaiting:  {GamePhasePreFlop},
		GamePhasePreFlop:  {GamePhaseFlop, GamePhaseShowdown, GamePhaseFinished},
		GamePhaseFlop:     {GamePhaseTurn, GamePhaseShowdown, GamePhaseFinished},
		GamePhaseTurn:     {GamePhaseRiver, GamePhaseShowdown, GamePhaseFinished},
		GamePhaseRiver:    {GamePhaseShowdown, GamePhaseFinished},
		GamePhaseShowdown: {GamePhaseFinished, GamePhasePreFlop},
		GamePhaseFinished: {GamePhaseWaiting, GamePhasePreFlop},
	}

	allowedPhases, exists := validTransitions[g.Phase]
	if !exists {
		return false
	}

	for _, allowed := range allowedPhases {
		if allowed == nextPhase {
			return true
		}
	}

	return false
}

// NextPhase - возвращает следующую фазу в нормальном течении игры
func (g *Game) NextPhase() GamePhase {
	switch g.Phase {
	case GamePhaseWaiting:
		return GamePhasePreFlop
	case GamePhasePreFlop:
		return GamePhaseFlop
	case GamePhaseFlop:
		return GamePhaseTurn
	case GamePhaseTurn:
		return GamePhaseRiver
	case GamePhaseRiver:
		return GamePhaseShowdown
	case GamePhaseShowdown:
		return GamePhaseFinished
	case GamePhaseFinished:
		return GamePhaseWaiting
	default:
		return GamePhaseWaiting
	}
}

// SetPhase - устанавливает новую фазу игры
func (g *Game) SetPhase(phase GamePhase) {
	g.Phase = phase
}

// === МЕТОДЫ ДЛЯ РАБОТЫ С БАНКОМ ===

// AddToPot - добавляет сумму к общему банку
func (g *Game) AddToPot(amount int) {
	g.Pot += amount
}

// ResetPot - сбрасывает банк в 0
func (g *Game) ResetPot() {
	g.Pot = 0
}

// GetTotalPot - возвращает общую сумму всех банков (main + side pots)
func (g *Game) GetTotalPot() int {
	total := g.Pot
	for _, sidePot := range g.SidePots {
		total += sidePot.Amount
	}
	return total
}

// === МЕТОДЫ ДЛЯ РАБОТЫ СО СТАВКАМИ ===

// SetCurrentBet - устанавливает текущую ставку
func (g *Game) SetCurrentBet(amount int) {
	g.CurrentBet = amount
}

// RaiseCurrentBet - увеличивает текущую ставку на указанную сумму
func (g *Game) RaiseCurrentBet(amount int) {
	g.CurrentBet += amount
}

// ResetCurrentBet - сбрасывает текущую ставку (начало нового раунда торговли)
func (g *Game) ResetCurrentBet() {
	g.CurrentBet = 0
}

// === МЕТОДЫ ДЛЯ РАБОТЫ С КАРТАМИ ===

// AddCommunityCard - добавляет карту на стол
func (g *Game) AddCommunityCard(card string) {
	g.CommunityCards = append(g.CommunityCards, card)
}

// AddCommunityCards - добавляет несколько карт на стол
func (g *Game) AddCommunityCards(cards []string) {
	g.CommunityCards = append(g.CommunityCards, cards...)
}

// GetCommunityCardsCount - возвращает количество общих карт на столе
func (g *Game) GetCommunityCardsCount() int {
	return len(g.CommunityCards)
}

// ClearCommunityCards - очищает общие карты (начало новой раздачи)
func (g *Game) ClearCommunityCards() {
	g.CommunityCards = []string{}
}

// === МЕТОДЫ ДЛЯ РАБОТЫ С РАУНДАМИ ===

// IncrementRound - увеличивает номер раунда (новая раздача)
func (g *Game) IncrementRound() {
	g.RoundNumber++
}

// === МЕТОДЫ ДЛЯ РАБОТЫ С ПОЗИЦИЯМИ ===

// SetDealerPosition - устанавливает позицию дилера
func (g *Game) SetDealerPosition(position int) {
	g.DealerPosition = position
}

// MoveDealerPosition - перемещает дилера на следующую позицию
func (g *Game) MoveDealerPosition(maxPlayers int) {
	g.DealerPosition = (g.DealerPosition + 1) % maxPlayers
}

// === ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ ===

// GetDuration - возвращает длительность игры
func (g *Game) GetDuration() time.Duration {
	if g.StartedAt == nil {
		return 0
	}
	return time.Since(*g.StartedAt)
}

// IsStarted - проверяет, началась ли игра
func (g *Game) IsStarted() bool {
	return g.StartedAt != nil
}

// === СТАТИЧЕСКИЕ ФУНКЦИИ ===

// ParseGamePhase - парсит строку в GamePhase
func ParseGamePhase(phase string) GamePhase {
	switch phase {
	case "waiting":
		return GamePhaseWaiting
	case "pre_flop":
		return GamePhasePreFlop
	case "flop":
		return GamePhaseFlop
	case "turn":
		return GamePhaseTurn
	case "river":
		return GamePhaseRiver
	case "showdown":
		return GamePhaseShowdown
	case "finished":
		return GamePhaseFinished
	default:
		return GamePhaseWaiting
	}
}

// IsValidGamePhase - проверяет, является ли строка валидной фазой
func IsValidGamePhase(phase string) bool {
	switch phase {
	case "waiting", "pre_flop", "flop", "turn", "river", "showdown", "finished":
		return true
	default:
		return false
	}
}

// GetAllGamePhases - возвращает все возможные фазы игры
func GetAllGamePhases() []GamePhase {
	return []GamePhase{
		GamePhaseWaiting,
		GamePhasePreFlop,
		GamePhaseFlop,
		GamePhaseTurn,
		GamePhaseRiver,
		GamePhaseShowdown,
		GamePhaseFinished,
	}
}
