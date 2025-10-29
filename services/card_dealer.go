package services

import (
	"encoding/json"
	"fmt"

	"poker-engine/storage"
	"poker-engine/utils"
)

// CardDealer - сервис для раздачи карт игрокам
type CardDealer struct {
	// redis - клиент для работы с Redis
	redis *storage.RedisClient

	// deckManager - менеджер колоды
	deckManager *DeckManager

	// gameStateService - сервис состояния игры
	gameStateService *GameStateService

	// logger - логгер для вывода сообщений
	logger *utils.Logger
}

// NewCardDealer - создает новый экземпляр CardDealer
func NewCardDealer(
	redis *storage.RedisClient,
	deckManager *DeckManager,
	gameStateService *GameStateService,
) *CardDealer {
	return &CardDealer{
		redis:            redis,
		deckManager:      deckManager,
		gameStateService: gameStateService,
		logger:           utils.NewLogger("CardDealer"),
	}
}

// DealCardsToPlayers - раздает карты всем игрокам в комнате
// В техасском холдеме каждый игрок получает 2 карты
func (cd *CardDealer) DealCardsToPlayers(clubID, roomID string) error {
	cd.logger.Infof("Начинаем раздачу карт в комнате %s:%s", clubID, roomID)

	// Шаг 1: Получаем список всех игроков в комнате
	playerIDs, err := cd.gameStateService.GetPlayerIDs(clubID, roomID)
	if err != nil {
		cd.logger.Errorf("Ошибка при получении списка игроков: %v", err)
		return fmt.Errorf("не удалось получить список игроков: %w", err)
	}

	if len(playerIDs) == 0 {
		cd.logger.Warning("Нет игроков для раздачи карт")
		return fmt.Errorf("нет игроков в комнате")
	}

	cd.logger.Infof("Найдено %d игроков для раздачи карт", len(playerIDs))

	// Шаг 2: Создаем и тасуем новую колоду
	deck := cd.deckManager.CreateAndShuffleDeck()
	cd.logger.Debugf("Колода создана и перетасована (%d карт)", len(deck))

	// Шаг 3: Сохраняем колоду в Redis
	err = cd.deckManager.SaveDeckToRedis(clubID, roomID, deck)
	if err != nil {
		return fmt.Errorf("не удалось сохранить колоду: %w", err)
	}

	// Шаг 4: Раздаем по 2 карты каждому игроку
	cardsPerPlayer := 2 // В техасском холдеме всегда 2 карты

	for _, userID := range playerIDs {
		// Берем 2 карты из колоды
		cards, err := cd.deckManager.DrawCards(clubID, roomID, cardsPerPlayer)
		if err != nil {
			cd.logger.Errorf("Ошибка при взятии карт для игрока %s: %v", userID, err)
			return fmt.Errorf("ошибка при взятии карт: %w", err)
		}

		if len(cards) != cardsPerPlayer {
			cd.logger.Errorf("Недостаточно карт в колоде для игрока %s", userID)
			return fmt.Errorf("недостаточно карт в колоде")
		}

		// Сохраняем карты игроку в Redis
		err = cd.savePlayerCards(clubID, roomID, userID, cards)
		if err != nil {
			cd.logger.Errorf("Ошибка при сохранении карт игрока %s: %v", userID, err)
			return fmt.Errorf("не удалось сохранить карты игроку: %w", err)
		}

		// Форматируем карты для красивого вывода в лог
		formattedCards := FormatCards(cards)
		cd.logger.Debugf("Игрок %s получил карты: %v", userID, formattedCards)
	}

	// Шаг 5: Проверяем сколько карт осталось в колоде
	remainingCards, _ := cd.deckManager.GetDeckSize(clubID, roomID)
	cd.logger.Infof("Раздача карт завершена. Роздано: %d игрокам × %d карт = %d карт. Осталось в колоде: %d",
		len(playerIDs), cardsPerPlayer, len(playerIDs)*cardsPerPlayer, remainingCards)

	return nil
}

// savePlayerCards - сохраняет карты игроку в Redis
func (cd *CardDealer) savePlayerCards(clubID, roomID, userID string, cards []string) error {
	// Получаем ключ для данных игрока
	playerKey := cd.redis.GetKeys().PlayerInfo(clubID, roomID, userID)

	// Конвертируем карты в JSON
	cardsJSON, err := json.Marshal(cards)
	if err != nil {
		return fmt.Errorf("ошибка при сериализации карт: %w", err)
	}

	// Сохраняем карты в Redis
	err = cd.redis.HSet(playerKey, "cards", string(cardsJSON))
	if err != nil {
		return fmt.Errorf("ошибка при сохранении карт в Redis: %w", err)
	}

	return nil
}

// DealCommunityCards - раздает общие карты на стол (флоп, терн, ривер)
// count: 3 для флопа, 1 для терна, 1 для ривера
func (cd *CardDealer) DealCommunityCards(clubID, roomID string, count int) ([]string, error) {
	cd.logger.Infof("Раздаем %d общих карт в комнате %s:%s", count, clubID, roomID)

	// Берем карты из колоды
	cards, err := cd.deckManager.DrawCards(clubID, roomID, count)
	if err != nil {
		cd.logger.Errorf("Ошибка при взятии общих карт: %v", err)
		return nil, fmt.Errorf("не удалось взять карты из колоды: %w", err)
	}

	if len(cards) != count {
		cd.logger.Errorf("Недостаточно карт в колоде (запрошено %d, получено %d)", count, len(cards))
		return nil, fmt.Errorf("недостаточно карт в колоде")
	}

	// Получаем текущие общие карты
	currentCards, err := cd.getCurrentCommunityCards(clubID, roomID)
	if err != nil {
		cd.logger.Errorf("Ошибка при получении текущих общих карт: %v", err)
		return nil, err
	}

	// Добавляем новые карты к существующим
	allCards := append(currentCards, cards...)

	// Сохраняем обновленный список в Redis
	err = cd.saveCommunityCards(clubID, roomID, allCards)
	if err != nil {
		cd.logger.Errorf("Ошибка при сохранении общих карт: %v", err)
		return nil, err
	}

	// Форматируем для красивого вывода
	formattedCards := FormatCards(cards)
	cd.logger.Infof("Общие карты добавлены: %v (всего на столе: %d)", formattedCards, len(allCards))

	return cards, nil
}

// getCurrentCommunityCards - получает текущие общие карты из Redis
func (cd *CardDealer) getCurrentCommunityCards(clubID, roomID string) ([]string, error) {
	gameKey := cd.redis.GetKeys().GameState(clubID, roomID)

	// Получаем поле community_cards
	cardsJSON, err := cd.redis.HGet(gameKey, "community_cards")
	if err != nil {
		return nil, err
	}

	// Если поле пустое или "[]", возвращаем пустой массив
	if cardsJSON == "" || cardsJSON == "[]" {
		return []string{}, nil
	}

	// Парсим JSON
	var cards []string
	err = json.Unmarshal([]byte(cardsJSON), &cards)
	if err != nil {
		return nil, fmt.Errorf("ошибка при парсинге общих карт: %w", err)
	}

	return cards, nil
}

// saveCommunityCards - сохраняет общие карты в Redis
func (cd *CardDealer) saveCommunityCards(clubID, roomID string, cards []string) error {
	gameKey := cd.redis.GetKeys().GameState(clubID, roomID)

	// Конвертируем в JSON
	cardsJSON, err := json.Marshal(cards)
	if err != nil {
		return fmt.Errorf("ошибка при сериализации общих карт: %w", err)
	}

	// Сохраняем в Redis
	err = cd.redis.HSet(gameKey, "community_cards", string(cardsJSON))
	if err != nil {
		return fmt.Errorf("ошибка при сохранении общих карт: %w", err)
	}

	return nil
}

// GetPlayerCards - получает карты игрока из Redis
func (cd *CardDealer) GetPlayerCards(clubID, roomID, userID string) ([]string, error) {
	playerKey := cd.redis.GetKeys().PlayerInfo(clubID, roomID, userID)

	// Получаем поле cards
	cardsJSON, err := cd.redis.HGet(playerKey, "cards")
	if err != nil {
		return nil, err
	}

	if cardsJSON == "" || cardsJSON == "[]" {
		return []string{}, nil
	}

	// Парсим JSON
	var cards []string
	err = json.Unmarshal([]byte(cardsJSON), &cards)
	if err != nil {
		return nil, fmt.Errorf("ошибка при парсинге карт игрока: %w", err)
	}

	return cards, nil
}

// ClearAllCards - очищает карты всех игроков и общие карты
// Используется в конце раздачи перед новой раздачей
func (cd *CardDealer) ClearAllCards(clubID, roomID string) error {
	cd.logger.Infof("Очищаем все карты в комнате %s:%s", clubID, roomID)

	// Получаем список всех игроков
	playerIDs, err := cd.gameStateService.GetPlayerIDs(clubID, roomID)
	if err != nil {
		return err
	}

	// Очищаем карты каждого игрока
	for _, userID := range playerIDs {
		err = cd.savePlayerCards(clubID, roomID, userID, []string{})
		if err != nil {
			cd.logger.Warningf("Не удалось очистить карты игрока %s: %v", userID, err)
		}
	}

	// Очищаем общие карты
	err = cd.saveCommunityCards(clubID, roomID, []string{})
	if err != nil {
		return err
	}

	// Удаляем колоду из Redis
	deckKey := cd.redis.GetKeys().RoomDeck(clubID, roomID)
	err = cd.redis.Del(deckKey)
	if err != nil {
		cd.logger.Warningf("Не удалось удалить колоду: %v", err)
	}

	cd.logger.Info("Все карты очищены")

	return nil
}

// BurnCard - "сжигает" одну карту (берет из колоды но не используется)
// В покере перед флопом, терном и ривером сжигается одна карта
func (cd *CardDealer) BurnCard(clubID, roomID string) error {
	card, err := cd.deckManager.DrawCard(clubID, roomID)
	if err != nil {
		return err
	}

	if card == "" {
		return fmt.Errorf("нет карт в колоде для сжигания")
	}

	cd.logger.Debugf("Карта %s сожжена", FormatCard(card))

	return nil
}
