package services

import (
	"math/rand"
	"time"

	"poker-engine/storage"
	"poker-engine/utils"
)

// DeckManager - сервис для работы с колодой карт
type DeckManager struct {
	// redis - клиент для работы с Redis
	redis *storage.RedisClient

	// logger - логгер для вывода сообщений
	logger *utils.Logger

	// rng - генератор случайных чисел для тасовки
	rng *rand.Rand
}

// NewDeckManager - создает новый экземпляр DeckManager
func NewDeckManager(redis *storage.RedisClient) *DeckManager {
	// Создаем источник случайных чисел с seed на основе текущего времени
	// Это гарантирует разную тасовку каждый раз
	source := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(source)

	return &DeckManager{
		redis:  redis,
		logger: utils.NewLogger("DeckManager"),
		rng:    rng,
	}
}

// Масти карт (Suits)
const (
	Hearts   = "H" // Червы (♥)
	Diamonds = "D" // Бубны (♦)
	Clubs    = "C" // Трефы (♣)
	Spades   = "S" // Пики (♠)
)

// Достоинства карт (Ranks)
const (
	Ace   = "A" // Туз
	Two   = "2" // Двойка
	Three = "3" // Тройка
	Four  = "4" // Четверка
	Five  = "5" // Пятерка
	Six   = "6" // Шестерка
	Seven = "7" // Семерка
	Eight = "8" // Восьмерка
	Nine  = "9" // Девятка
	Ten   = "T" // Десятка
	Jack  = "J" // Валет
	Queen = "Q" // Дама
	King  = "K" // Король
)

// CreateDeck - создает новую колоду из 52 карт
// Возвращает массив строк вида: ["AH", "2H", "3H", ..., "KS"]
func (dm *DeckManager) CreateDeck() []string {
	// Массив мастей (4 масти)
	suits := []string{Hearts, Diamonds, Clubs, Spades}

	// Массив достоинств (13 достоинств)
	ranks := []string{
		Ace, Two, Three, Four, Five, Six, Seven,
		Eight, Nine, Ten, Jack, Queen, King,
	}

	// Создаем пустой массив для колоды
	// 4 масти × 13 достоинств = 52 карты
	deck := make([]string, 0, 52)

	// Проходим по всем мастям
	for _, suit := range suits {
		// Для каждой масти проходим по всем достоинствам
		for _, rank := range ranks {
			// Формируем карту: Достоинство + Масть
			// Например: "A" + "H" = "AH" (Туз червей)
			card := rank + suit
			deck = append(deck, card)
		}
	}

	dm.logger.Debugf("Создана новая колода из %d карт", len(deck))

	return deck
}

// ShuffleDeck - тасует колоду карт (Fisher-Yates shuffle)
// Это самый эффективный алгоритм тасовки - O(n)
// Гарантирует равномерное распределение
func (dm *DeckManager) ShuffleDeck(deck []string) []string {
	// Создаем копию колоды, чтобы не изменять оригинал
	shuffled := make([]string, len(deck))
	copy(shuffled, deck)

	// Алгоритм Fisher-Yates:
	// Идем с конца к началу
	for i := len(shuffled) - 1; i > 0; i-- {
		// Выбираем случайную позицию от 0 до i
		j := dm.rng.Intn(i + 1)

		// Меняем местами карты на позициях i и j
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	dm.logger.Debugf("Колода перетасована")

	return shuffled
}

// CreateAndShuffleDeck - создает и сразу тасует колоду (удобный метод)
func (dm *DeckManager) CreateAndShuffleDeck() []string {
	deck := dm.CreateDeck()
	return dm.ShuffleDeck(deck)
}

// SaveDeckToRedis - сохраняет колоду в Redis для комнаты
// Сохраняется как список (LIST), карты берутся с конца (RPOP)
func (dm *DeckManager) SaveDeckToRedis(clubID, roomID string, deck []string) error {
	// Получаем ключ для колоды
	deckKey := dm.redis.GetKeys().RoomDeck(clubID, roomID)

	// Удаляем старую колоду если была
	err := dm.redis.Del(deckKey)
	if err != nil {
		dm.logger.Errorf("Ошибка при удалении старой колоды: %v", err)
		return err
	}

	// Сохраняем новую колоду
	// Конвертируем []string в []interface{} для Redis
	deckInterface := make([]interface{}, len(deck))
	for i, card := range deck {
		deckInterface[i] = card
	}

	// Добавляем все карты в список (LIST) в Redis
	err = dm.redis.RPush(deckKey, deckInterface...)
	if err != nil {
		dm.logger.Errorf("Ошибка при сохранении колоды в Redis: %v", err)
		return err
	}

	dm.logger.Infof("Колода из %d карт сохранена в Redis для комнаты %s:%s", len(deck), clubID, roomID)

	return nil
}

// DrawCard - берет одну карту из колоды (с конца списка)
// Возвращает карту или пустую строку если колода пуста
func (dm *DeckManager) DrawCard(clubID, roomID string) (string, error) {
	deckKey := dm.redis.GetKeys().RoomDeck(clubID, roomID)

	// RPOP - берет и удаляет последний элемент списка
	card, err := dm.redis.GetClient().RPop(dm.redis.GetContext(), deckKey).Result()
	if err != nil {
		// Если колода пуста, Redis вернет ошибку "redis: nil"
		if err.Error() == "redis: nil" {
			dm.logger.Warning("Колода пуста!")
			return "", nil
		}
		dm.logger.Errorf("Ошибка при взятии карты из колоды: %v", err)
		return "", err
	}

	return card, nil
}

// DrawCards - берет несколько карт из колоды
func (dm *DeckManager) DrawCards(clubID, roomID string, count int) ([]string, error) {
	cards := make([]string, 0, count)

	for i := 0; i < count; i++ {
		card, err := dm.DrawCard(clubID, roomID)
		if err != nil {
			return nil, err
		}
		if card == "" {
			// Колода закончилась
			dm.logger.Warningf("Колода закончилась после %d карт из %d", i, count)
			break
		}
		cards = append(cards, card)
	}

	return cards, nil
}

// GetDeckSize - возвращает количество оставшихся карт в колоде
func (dm *DeckManager) GetDeckSize(clubID, roomID string) (int64, error) {
	deckKey := dm.redis.GetKeys().RoomDeck(clubID, roomID)
	size, err := dm.redis.LLen(deckKey)
	if err != nil {
		dm.logger.Errorf("Ошибка при получении размера колоды: %v", err)
		return 0, err
	}
	return size, nil
}

// === ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ ===

// ParseCard - разбирает карту на достоинство и масть
// Например: "AH" → достоинство="A", масть="H"
func ParseCard(card string) (rank string, suit string) {
	if len(card) != 2 {
		return "", ""
	}
	return string(card[0]), string(card[1])
}

// FormatCard - форматирует карту для отображения
// Например: "AH" → "A♥"
func FormatCard(card string) string {
	if len(card) != 2 {
		return card
	}

	rank := string(card[0])
	suit := string(card[1])

	// Конвертируем буквы в символы мастей
	suitSymbol := suit
	switch suit {
	case Hearts:
		suitSymbol = "♥"
	case Diamonds:
		suitSymbol = "♦"
	case Clubs:
		suitSymbol = "♣"
	case Spades:
		suitSymbol = "♠"
	}

	return rank + suitSymbol
}

// FormatCards - форматирует массив карт для отображения
func FormatCards(cards []string) []string {
	formatted := make([]string, len(cards))
	for i, card := range cards {
		formatted[i] = FormatCard(card)
	}
	return formatted
}

// IsValidCard - проверяет, является ли строка валидной картой
func IsValidCard(card string) bool {
	if len(card) != 2 {
		return false
	}

	rank := string(card[0])
	suit := string(card[1])

	// Проверяем достоинство
	validRanks := []string{Ace, Two, Three, Four, Five, Six, Seven, Eight, Nine, Ten, Jack, Queen, King}
	validRank := false
	for _, r := range validRanks {
		if rank == r {
			validRank = true
			break
		}
	}
	if !validRank {
		return false
	}

	// Проверяем масть
	validSuits := []string{Hearts, Diamonds, Clubs, Spades}
	validSuit := false
	for _, s := range validSuits {
		if suit == s {
			validSuit = true
			break
		}
	}

	return validSuit
}
