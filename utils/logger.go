package utils

import (
	"fmt"
	"log"
	"time"
)

// Logger - структура для логирования с красивым форматированием
type Logger struct {
	// prefix - префикс для всех сообщений (например: "[GameEngine]")
	prefix string

	// showTimestamp - показывать ли timestamp в логах
	showTimestamp bool
}

// NewLogger - создает новый логгер с заданным префиксом
func NewLogger(prefix string) *Logger {
	return &Logger{
		prefix:        prefix,
		showTimestamp: true,
	}
}

// getTimestamp - возвращает текущее время в формате HH:MM:SS
func (l *Logger) getTimestamp() string {
	if !l.showTimestamp {
		return ""
	}
	return time.Now().Format("15:04:05")
}

// formatMessage - форматирует сообщение с префиксом и временем
func (l *Logger) formatMessage(emoji, level, message string) string {
	timestamp := l.getTimestamp()
	if timestamp != "" {
		return fmt.Sprintf("%s [%s] %s %s: %s", emoji, timestamp, l.prefix, level, message)
	}
	return fmt.Sprintf("%s %s %s: %s", emoji, l.prefix, level, message)
}

// Info - логирует информационное сообщение (обычное)
func (l *Logger) Info(message string) {
	log.Println(l.formatMessage("ℹ️", "INFO", message))
}

// Infof - логирует информационное сообщение с форматированием (как printf)
func (l *Logger) Infof(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	l.Info(message)
}

// Success - логирует успешное действие (зеленая галочка)
func (l *Logger) Success(message string) {
	log.Println(l.formatMessage("✅", "SUCCESS", message))
}

// Successf - логирует успешное действие с форматированием
func (l *Logger) Successf(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	l.Success(message)
}

// Warning - логирует предупреждение (желтый треугольник)
func (l *Logger) Warning(message string) {
	log.Println(l.formatMessage("⚠️", "WARNING", message))
}

// Warningf - логирует предупреждение с форматированием
func (l *Logger) Warningf(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	l.Warning(message)
}

// Error - логирует ошибку (красный крестик)
func (l *Logger) Error(message string) {
	log.Println(l.formatMessage("❌", "ERROR", message))
}

// Errorf - логирует ошибку с форматированием
func (l *Logger) Errorf(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	l.Error(message)
}

// Debug - логирует отладочное сообщение (для разработки)
func (l *Logger) Debug(message string) {
	log.Println(l.formatMessage("🔍", "DEBUG", message))
}

// Debugf - логирует отладочное сообщение с форматированием
func (l *Logger) Debugf(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	l.Debug(message)
}

// === Специализированные методы для игровых событий ===

// GameStarted - логирует запуск игры
func (l *Logger) GameStarted(clubID, roomID, gameID string, playersCount int) {
	log.Println(l.formatMessage("🎯", "GAME",
		fmt.Sprintf("Запуск игры | Club: %s | Room: %s | GameID: %s | Игроков: %d | Фаза: pre_flop",
			clubID, roomID, gameID, playersCount)))
}

// GameStopped - логирует остановку игры
func (l *Logger) GameStopped(clubID, roomID, previousPhase string, reason string) {
	log.Println(l.formatMessage("🛑", "GAME",
		fmt.Sprintf("Остановка игры | Club: %s | Room: %s | Фаза была: %s | Причина: %s",
			clubID, roomID, previousPhase, reason)))
}

// GameInProgress - логирует текущее состояние игры
func (l *Logger) GameInProgress(clubID, roomID, phase string, playersCount int) {
	log.Println(l.formatMessage("🎮", "GAME",
		fmt.Sprintf("Игра идет | Club: %s | Room: %s | Фаза: %s | Игроков: %d",
			clubID, roomID, phase, playersCount)))
}

// RoomMonitoring - логирует начало мониторинга комнат
func (l *Logger) RoomMonitoring() {
	log.Println(l.formatMessage("🔄", "MONITOR", "Проверка всех активных комнат..."))
}

// RoomFound - логирует найденную комнату
func (l *Logger) RoomFound(clubID, roomID string, playersCount int) {
	log.Println(l.formatMessage("🏠", "ROOM",
		fmt.Sprintf("Комната найдена | Club: %s | Room: %s | Игроков: %d",
			clubID, roomID, playersCount)))
}

// PlayerJoined - логирует присоединение игрока
func (l *Logger) PlayerJoined(clubID, roomID, userID string, playersCount int) {
	log.Println(l.formatMessage("👤", "PLAYER",
		fmt.Sprintf("Игрок присоединился | Club: %s | Room: %s | UserID: %s | Всего игроков: %d",
			clubID, roomID, userID, playersCount)))
}

// PlayerLeft - логирует выход игрока
func (l *Logger) PlayerLeft(clubID, roomID, userID string, playersCount int) {
	log.Println(l.formatMessage("👋", "PLAYER",
		fmt.Sprintf("Игрок вышел | Club: %s | Room: %s | UserID: %s | Осталось игроков: %d",
			clubID, roomID, userID, playersCount)))
}

// RedisConnected - логирует успешное подключение к Redis
func (l *Logger) RedisConnected(addr string) {
	log.Println(l.formatMessage("✅", "REDIS",
		fmt.Sprintf("Подключение к Redis успешно | Адрес: %s", addr)))
}

// RedisError - логирует ошибку Redis
func (l *Logger) RedisError(operation string, err error) {
	log.Println(l.formatMessage("❌", "REDIS",
		fmt.Sprintf("Ошибка при операции '%s': %v", operation, err)))
}

// RedisReconnecting - логирует попытку переподключения к Redis
func (l *Logger) RedisReconnecting(attempt int) {
	log.Println(l.formatMessage("🔄", "REDIS",
		fmt.Sprintf("Попытка переподключения #%d...", attempt)))
}

// EngineStarted - логирует запуск движка
func (l *Logger) EngineStarted() {
	l.printBanner()
	log.Println(l.formatMessage("🚀", "ENGINE", "Game Engine запущен и готов к работе"))
}

// EngineStopped - логирует остановку движка
func (l *Logger) EngineStopped() {
	log.Println(l.formatMessage("🛑", "ENGINE", "Game Engine остановлен"))
}

// EngineShuttingDown - логирует начало процесса остановки
func (l *Logger) EngineShuttingDown() {
	log.Println(l.formatMessage("⏳", "ENGINE", "Получен сигнал завершения, останавливаем движок..."))
}

// printBanner - печатает красивый баннер при запуске
func (l *Logger) printBanner() {
	banner := `
╔════════════════════════════════════════════════════════╗
║                                                        ║
║              🎲  POKER GAME ENGINE  🎲                 ║
║                                                        ║
║              Версия: 1.0.0                            ║
║              Go High-Performance Game Server          ║
║                                                        ║
╚════════════════════════════════════════════════════════╝
`
	fmt.Println(banner)
}

// PrintSeparator - печатает разделитель для лучшей читаемости логов
func (l *Logger) PrintSeparator() {
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// PrintHeader - печатает заголовок секции
func (l *Logger) PrintHeader(title string) {
	l.PrintSeparator()
	log.Printf("  📋 %s", title)
	l.PrintSeparator()
}

// === Глобальный логгер по умолчанию ===

var defaultLogger = NewLogger("GameEngine")

// Глобальные функции для удобства использования без создания логгера

func Info(message string) {
	defaultLogger.Info(message)
}

func Infof(format string, args ...interface{}) {
	defaultLogger.Infof(format, args...)
}

func Success(message string) {
	defaultLogger.Success(message)
}

func Successf(format string, args ...interface{}) {
	defaultLogger.Successf(format, args...)
}

func Warning(message string) {
	defaultLogger.Warning(message)
}

func Warningf(format string, args ...interface{}) {
	defaultLogger.Warningf(format, args...)
}

func Error(message string) {
	defaultLogger.Error(message)
}

func Errorf(format string, args ...interface{}) {
	defaultLogger.Errorf(format, args...)
}

func Debug(message string) {
	defaultLogger.Debug(message)
}

func Debugf(format string, args ...interface{}) {
	defaultLogger.Debugf(format, args...)
}
