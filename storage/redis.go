package storage

import (
	"context"
	"fmt"
	"time"

	"poker-engine/config"
	"poker-engine/utils"

	"github.com/redis/go-redis/v9"
)

// RedisClient - обертка над Redis клиентом с дополнительными возможностями
type RedisClient struct {
	// client - основной Redis клиент
	client *redis.Client

	// ctx - контекст для всех операций
	ctx context.Context

	// config - конфигурация Redis
	config *config.RedisConfig

	// logger - логгер для вывода сообщений
	logger *utils.Logger

	// keys - генератор ключей Redis
	keys *Keys

	// isConnected - флаг состояния подключения
	isConnected bool
}

// NewRedisClient - создает новый Redis клиент
func NewRedisClient(ctx context.Context, cfg *config.RedisConfig) (*RedisClient, error) {
	logger := utils.NewLogger("Redis")

	// Создаем Redis клиент с настройками из конфигурации
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,         // Адрес Redis сервера
		Password:     cfg.Password,     // Пароль (может быть пустым)
		DB:           cfg.DB,           // Номер базы данных
		PoolSize:     cfg.PoolSize,     // Размер пула соединений
		MinIdleConns: cfg.MinIdleConns, // Минимум idle соединений
		MaxRetries:   cfg.MaxRetries,   // Максимум повторных попыток
		DialTimeout:  cfg.DialTimeout,  // Таймаут подключения
		ReadTimeout:  cfg.ReadTimeout,  // Таймаут чтения
		WriteTimeout: cfg.WriteTimeout, // Таймаут записи
	})

	rc := &RedisClient{
		client:      client,
		ctx:         ctx,
		config:      cfg,
		logger:      logger,
		keys:        NewKeys(),
		isConnected: false,
	}

	// Проверяем подключение
	if err := rc.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	rc.isConnected = true
	logger.RedisConnected(cfg.Addr)

	return rc, nil
}

// Ping - проверяет соединение с Redis
func (r *RedisClient) Ping() error {
	_, err := r.client.Ping(r.ctx).Result()
	if err != nil {
		r.isConnected = false
		r.logger.RedisError("ping", err)
		return err
	}
	r.isConnected = true
	return nil
}

// IsConnected - возвращает статус подключения
func (r *RedisClient) IsConnected() bool {
	return r.isConnected
}

// GetClient - возвращает базовый Redis клиент (для прямого доступа если нужно)
func (r *RedisClient) GetClient() *redis.Client {
	return r.client
}

// GetKeys - возвращает генератор ключей
func (r *RedisClient) GetKeys() *Keys {
	return r.keys
}

// GetContext - возвращает контекст клиента
func (r *RedisClient) GetContext() context.Context {
	return r.ctx
}

// Close - закрывает соединение с Redis
func (r *RedisClient) Close() error {
	r.logger.Info("Закрытие соединения с Redis...")
	err := r.client.Close()
	if err != nil {
		r.logger.RedisError("close", err)
		return err
	}
	r.isConnected = false
	r.logger.Success("Соединение с Redis закрыто")
	return nil
}

// === МЕТОДЫ ДЛЯ РАБОТЫ С ДАННЫМИ ===

// Get - получает значение по ключу (STRING)
func (r *RedisClient) Get(key string) (string, error) {
	val, err := r.client.Get(r.ctx, key).Result()
	if err == redis.Nil {
		// Ключ не существует - это не ошибка, просто возвращаем пустую строку
		return "", nil
	}
	if err != nil {
		r.logger.RedisError(fmt.Sprintf("GET %s", key), err)
		return "", err
	}
	return val, nil
}

// Set - устанавливает значение по ключу (STRING)
func (r *RedisClient) Set(key string, value interface{}, expiration time.Duration) error {
	err := r.client.Set(r.ctx, key, value, expiration).Err()
	if err != nil {
		r.logger.RedisError(fmt.Sprintf("SET %s", key), err)
		return err
	}
	return nil
}

// Del - удаляет ключ(и)
func (r *RedisClient) Del(keys ...string) error {
	err := r.client.Del(r.ctx, keys...).Err()
	if err != nil {
		r.logger.RedisError(fmt.Sprintf("DEL %v", keys), err)
		return err
	}
	return nil
}

// Exists - проверяет существование ключа
func (r *RedisClient) Exists(key string) (bool, error) {
	val, err := r.client.Exists(r.ctx, key).Result()
	if err != nil {
		r.logger.RedisError(fmt.Sprintf("EXISTS %s", key), err)
		return false, err
	}
	return val > 0, nil
}

// === HASH ОПЕРАЦИИ ===

// HGet - получает значение поля из hash
func (r *RedisClient) HGet(key, field string) (string, error) {
	val, err := r.client.HGet(r.ctx, key, field).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		r.logger.RedisError(fmt.Sprintf("HGET %s %s", key, field), err)
		return "", err
	}
	return val, nil
}

// HGetAll - получает все поля и значения из hash
func (r *RedisClient) HGetAll(key string) (map[string]string, error) {
	val, err := r.client.HGetAll(r.ctx, key).Result()
	if err != nil {
		r.logger.RedisError(fmt.Sprintf("HGETALL %s", key), err)
		return nil, err
	}
	return val, nil
}

// HSet - устанавливает значение поля в hash
func (r *RedisClient) HSet(key string, field string, value interface{}) error {
	err := r.client.HSet(r.ctx, key, field, value).Err()
	if err != nil {
		r.logger.RedisError(fmt.Sprintf("HSET %s %s", key, field), err)
		return err
	}
	return nil
}

// HMSet - устанавливает несколько полей в hash
func (r *RedisClient) HMSet(key string, values map[string]interface{}) error {
	err := r.client.HMSet(r.ctx, key, values).Err()
	if err != nil {
		r.logger.RedisError(fmt.Sprintf("HMSET %s", key), err)
		return err
	}
	return nil
}

// === SET ОПЕРАЦИИ ===

// SAdd - добавляет элемент(ы) в set
func (r *RedisClient) SAdd(key string, members ...interface{}) error {
	err := r.client.SAdd(r.ctx, key, members...).Err()
	if err != nil {
		r.logger.RedisError(fmt.Sprintf("SADD %s", key), err)
		return err
	}
	return nil
}

// SRem - удаляет элемент(ы) из set
func (r *RedisClient) SRem(key string, members ...interface{}) error {
	err := r.client.SRem(r.ctx, key, members...).Err()
	if err != nil {
		r.logger.RedisError(fmt.Sprintf("SREM %s", key), err)
		return err
	}
	return nil
}

// SMembers - получает все элементы из set
func (r *RedisClient) SMembers(key string) ([]string, error) {
	val, err := r.client.SMembers(r.ctx, key).Result()
	if err != nil {
		r.logger.RedisError(fmt.Sprintf("SMEMBERS %s", key), err)
		return nil, err
	}
	return val, nil
}

// SCard - получает количество элементов в set
func (r *RedisClient) SCard(key string) (int64, error) {
	val, err := r.client.SCard(r.ctx, key).Result()
	if err != nil {
		r.logger.RedisError(fmt.Sprintf("SCARD %s", key), err)
		return 0, err
	}
	return val, nil
}

// SIsMember - проверяет, находится ли элемент в set
func (r *RedisClient) SIsMember(key string, member interface{}) (bool, error) {
	val, err := r.client.SIsMember(r.ctx, key, member).Result()
	if err != nil {
		r.logger.RedisError(fmt.Sprintf("SISMEMBER %s", key), err)
		return false, err
	}
	return val, nil
}

// === SORTED SET ОПЕРАЦИИ ===

// ZAdd - добавляет элемент в sorted set с указанным score
func (r *RedisClient) ZAdd(key string, score float64, member interface{}) error {
	err := r.client.ZAdd(r.ctx, key, redis.Z{
		Score:  score,
		Member: member,
	}).Err()
	if err != nil {
		r.logger.RedisError(fmt.Sprintf("ZADD %s", key), err)
		return err
	}
	return nil
}

// ZRem - удаляет элемент из sorted set
func (r *RedisClient) ZRem(key string, members ...interface{}) error {
	err := r.client.ZRem(r.ctx, key, members...).Err()
	if err != nil {
		r.logger.RedisError(fmt.Sprintf("ZREM %s", key), err)
		return err
	}
	return nil
}

// ZRange - получает элементы из sorted set по диапазону индексов
// start=0, stop=-1 возвращает все элементы
func (r *RedisClient) ZRange(key string, start, stop int64) ([]string, error) {
	val, err := r.client.ZRange(r.ctx, key, start, stop).Result()
	if err != nil {
		r.logger.RedisError(fmt.Sprintf("ZRANGE %s %d %d", key, start, stop), err)
		return nil, err
	}
	return val, nil
}

// ZCard - получает количество элементов в sorted set
func (r *RedisClient) ZCard(key string) (int64, error) {
	val, err := r.client.ZCard(r.ctx, key).Result()
	if err != nil {
		r.logger.RedisError(fmt.Sprintf("ZCARD %s", key), err)
		return 0, err
	}
	return val, nil
}

// === LIST ОПЕРАЦИИ ===

// LPush - добавляет элемент в начало списка
func (r *RedisClient) LPush(key string, values ...interface{}) error {
	err := r.client.LPush(r.ctx, key, values...).Err()
	if err != nil {
		r.logger.RedisError(fmt.Sprintf("LPUSH %s", key), err)
		return err
	}
	return nil
}

// RPush - добавляет элемент в конец списка
func (r *RedisClient) RPush(key string, values ...interface{}) error {
	err := r.client.RPush(r.ctx, key, values...).Err()
	if err != nil {
		r.logger.RedisError(fmt.Sprintf("RPUSH %s", key), err)
		return err
	}
	return nil
}

// LRange - получает элементы из списка по диапазону индексов
func (r *RedisClient) LRange(key string, start, stop int64) ([]string, error) {
	val, err := r.client.LRange(r.ctx, key, start, stop).Result()
	if err != nil {
		r.logger.RedisError(fmt.Sprintf("LRANGE %s %d %d", key, start, stop), err)
		return nil, err
	}
	return val, nil
}

// LLen - получает длину списка
func (r *RedisClient) LLen(key string) (int64, error) {
	val, err := r.client.LLen(r.ctx, key).Result()
	if err != nil {
		r.logger.RedisError(fmt.Sprintf("LLEN %s", key), err)
		return 0, err
	}
	return val, nil
}

// === СКАНИРОВАНИЕ ===

// Scan - сканирует ключи по паттерну
// Возвращает итератор для последовательного чтения ключей
func (r *RedisClient) Scan(pattern string) *redis.ScanIterator {
	return r.client.Scan(r.ctx, 0, pattern, 0).Iterator()
}

// Keys - получает все ключи по паттерну (НЕ рекомендуется для продакшена!)
// Используй Scan для больших объемов данных
func (r *RedisClient) Keys(pattern string) ([]string, error) {
	val, err := r.client.Keys(r.ctx, pattern).Result()
	if err != nil {
		r.logger.RedisError(fmt.Sprintf("KEYS %s", pattern), err)
		return nil, err
	}
	return val, nil
}

// === PIPELINE ===

// Pipeline - создает новый pipeline для группировки команд
// Pipeline позволяет выполнить несколько команд атомарно и эффективно
func (r *RedisClient) Pipeline() redis.Pipeliner {
	return r.client.Pipeline()
}

// === ТРАНЗАКЦИИ ===

// TxPipeline - создает транзакционный pipeline
func (r *RedisClient) TxPipeline() redis.Pipeliner {
	return r.client.TxPipeline()
}

// === СЛУЖЕБНЫЕ МЕТОДЫ ===

// Reconnect - пытается переподключиться к Redis
func (r *RedisClient) Reconnect(maxAttempts int) error {
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		r.logger.RedisReconnecting(attempt)

		if err := r.Ping(); err == nil {
			r.logger.Success("Переподключение к Redis успешно")
			return nil
		}

		// Экспоненциальная задержка между попытками
		time.Sleep(time.Duration(attempt) * time.Second)
	}

	return fmt.Errorf("failed to reconnect after %d attempts", maxAttempts)
}

// HealthCheck - проверяет здоровье соединения
func (r *RedisClient) HealthCheck() error {
	if !r.isConnected {
		return fmt.Errorf("redis client is not connected")
	}

	return r.Ping()
}
