package config

import (
	"os"
	"strconv"
	"time"
)

// Config - главная структура конфигурации приложения
// Содержит все настройки для работы игрового движка
type Config struct {
	// Redis настройки подключения
	Redis RedisConfig

	// Engine настройки работы движка
	Engine EngineConfig
}

// RedisConfig - настройки подключения к Redis
type RedisConfig struct {
	// Addr - адрес Redis сервера (например: "localhost:6379")
	Addr string

	// Password - пароль для подключения (пустая строка если нет пароля)
	Password string

	// DB - номер базы данных Redis (0-15, обычно 0)
	DB int

	// PoolSize - размер пула соединений (количество одновременных подключений)
	PoolSize int

	// MinIdleConns - минимальное количество idle соединений в пуле
	MinIdleConns int

	// MaxRetries - максимальное количество повторных попыток при ошибке
	MaxRetries int

	// DialTimeout - таймаут подключения к Redis
	DialTimeout time.Duration

	// ReadTimeout - таймаут чтения данных
	ReadTimeout time.Duration

	// WriteTimeout - таймаут записи данных
	WriteTimeout time.Duration
}

// EngineConfig - настройки работы игрового движка
type EngineConfig struct {
	// CheckInterval - интервал проверки комнат (как часто сканировать Redis)
	CheckInterval time.Duration

	// ShutdownTimeout - таймаут для graceful shutdown
	ShutdownTimeout time.Duration

	// MinPlayersToStart - минимальное количество игроков для старта игры
	MinPlayersToStart int

	// MaxPlayersPerRoom - максимальное количество игроков в комнате
	MaxPlayersPerRoom int
}

// Load - загружает конфигурацию из переменных окружения с дефолтными значениями
// Если переменная окружения не установлена, используется дефолтное значение
func Load() *Config {
	return &Config{
		Redis: RedisConfig{
			// Адрес Redis: по умолчанию localhost:6379
			// Можно переопределить через ENV: REDIS_ADDR=your-redis:6379
			Addr: getEnv("REDIS_ADDR", "zxnxk365.redis.tools:10364"),

			// Пароль: по умолчанию пустой
			// Можно переопределить через ENV: REDIS_PASSWORD=your_password
			Password: getEnv("REDIS_PASSWORD", "yX4hdF8eqN"),

			// Номер базы данных: по умолчанию 0
			// Можно переопределить через ENV: REDIS_DB=1
			DB: getEnvAsInt("REDIS_DB", 0),

			// Размер пула соединений: по умолчанию 10
			// Увеличь если много комнат (100+ активных)
			PoolSize: getEnvAsInt("REDIS_POOL_SIZE", 10),

			// Минимум idle соединений: по умолчанию 5
			MinIdleConns: getEnvAsInt("REDIS_MIN_IDLE_CONNS", 5),

			// Максимум повторных попыток: по умолчанию 3
			MaxRetries: getEnvAsInt("REDIS_MAX_RETRIES", 3),

			// Таймаут подключения: по умолчанию 5 секунд
			DialTimeout: getEnvAsDuration("REDIS_DIAL_TIMEOUT", 5*time.Second),

			// Таймаут чтения: по умолчанию 3 секунды
			ReadTimeout: getEnvAsDuration("REDIS_READ_TIMEOUT", 3*time.Second),

			// Таймаут записи: по умолчанию 3 секунды
			WriteTimeout: getEnvAsDuration("REDIS_WRITE_TIMEOUT", 3*time.Second),
		},

		Engine: EngineConfig{
			// Интервал проверки комнат: по умолчанию 2 секунды
			// Уменьши до 1 секунды для более быстрой реакции
			CheckInterval: getEnvAsDuration("ENGINE_CHECK_INTERVAL", 2*time.Second),

			// Таймаут для graceful shutdown: по умолчанию 10 секунд
			ShutdownTimeout: getEnvAsDuration("ENGINE_SHUTDOWN_TIMEOUT", 10*time.Second),

			// Минимум игроков для старта: по умолчанию 2
			MinPlayersToStart: getEnvAsInt("ENGINE_MIN_PLAYERS", 2),

			// Максимум игроков в комнате: по умолчанию 9
			MaxPlayersPerRoom: getEnvAsInt("ENGINE_MAX_PLAYERS", 9),
		},
	}
}

// getEnv - получает значение переменной окружения или возвращает дефолтное
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvAsInt - получает переменную окружения как целое число
// Если не удается преобразовать или переменная не задана - возвращает дефолт
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

// getEnvAsDuration - получает переменную окружения как time.Duration
// Формат: "5s", "2m", "1h" и т.д.
// Если не удается распарсить или переменная не задана - возвращает дефолт
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

// Validate - проверяет корректность конфигурации
// Возвращает ошибку если какие-то параметры некорректны
func (c *Config) Validate() error {
	// Проверяем, что адрес Redis не пустой
	if c.Redis.Addr == "" {
		return ErrInvalidRedisAddr
	}

	// Проверяем, что номер БД в допустимом диапазоне (0-15 для стандартного Redis)
	if c.Redis.DB < 0 || c.Redis.DB > 15 {
		return ErrInvalidRedisDB
	}

	// Проверяем, что интервал проверки комнат больше 0
	if c.Engine.CheckInterval <= 0 {
		return ErrInvalidCheckInterval
	}

	// Проверяем, что минимум игроков >= 2 (покер не играется с 1 игроком)
	if c.Engine.MinPlayersToStart < 2 {
		return ErrInvalidMinPlayers
	}

	// Всё корректно
	return nil
}

// Ошибки валидации (определяем как константы для переиспользования)
var (
	ErrInvalidRedisAddr     = NewConfigError("redis address cannot be empty")
	ErrInvalidRedisDB       = NewConfigError("redis DB must be between 0 and 15")
	ErrInvalidCheckInterval = NewConfigError("check interval must be greater than 0")
	ErrInvalidMinPlayers    = NewConfigError("minimum players must be at least 2")
)

// ConfigError - кастомный тип ошибки конфигурации
type ConfigError struct {
	message string
}

func (e *ConfigError) Error() string {
	return "config error: " + e.message
}

// NewConfigError - создает новую ошибку конфигурации
func NewConfigError(message string) *ConfigError {
	return &ConfigError{message: message}
}
