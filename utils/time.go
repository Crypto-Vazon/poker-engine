package utils

import (
	"time"
)

// GetCurrentTimestamp - возвращает текущее время как Unix timestamp (секунды с 1970 года)
// Используется для записи времени действий в Redis
// Пример: 1730123456
func GetCurrentTimestamp() int64 {
	return time.Now().Unix()
}

// GetCurrentTimestampMillis - возвращает текущее время в миллисекундах
// Полезно для более точных измерений времени
// Пример: 1730123456789
func GetCurrentTimestampMillis() int64 {
	return time.Now().UnixMilli()
}

// GetISO8601Time - возвращает текущее время в формате ISO 8601
// Формат: 2025-10-28T14:30:00Z
// Используется для хранения читаемых дат в Redis
func GetISO8601Time() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// GetISO8601TimeLocal - возвращает текущее время в формате ISO 8601 (локальное время)
// Формат: 2025-10-28T16:30:00+02:00
func GetISO8601TimeLocal() string {
	return time.Now().Format(time.RFC3339)
}

// ParseISO8601 - парсит строку в формате ISO 8601 в time.Time
// Возвращает ошибку если формат некорректный
func ParseISO8601(timeStr string) (time.Time, error) {
	return time.Parse(time.RFC3339, timeStr)
}

// ParseTimestamp - конвертирует Unix timestamp в time.Time
// Принимает секунды с 1970 года
func ParseTimestamp(timestamp int64) time.Time {
	return time.Unix(timestamp, 0)
}

// ParseTimestampMillis - конвертирует Unix timestamp (миллисекунды) в time.Time
func ParseTimestampMillis(timestampMs int64) time.Time {
	return time.UnixMilli(timestampMs)
}

// FormatDuration - форматирует duration в читаемый вид
// Примеры:
//   - 1h30m -> "1 час 30 минут"
//   - 45s -> "45 секунд"
//   - 2m30s -> "2 минуты 30 секунд"
func FormatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		if minutes > 0 {
			return formatTime(hours, "час", "часа", "часов") + " " +
				formatTime(minutes, "минута", "минуты", "минут")
		}
		return formatTime(hours, "час", "часа", "часов")
	}

	if minutes > 0 {
		if seconds > 0 {
			return formatTime(minutes, "минута", "минуты", "минут") + " " +
				formatTime(seconds, "секунда", "секунды", "секунд")
		}
		return formatTime(minutes, "минута", "минуты", "минут")
	}

	return formatTime(seconds, "секунда", "секунды", "секунд")
}

// formatTime - вспомогательная функция для правильного склонения русских слов
// Примеры:
//   - 1 час, 2 часа, 5 часов
//   - 1 минута, 2 минуты, 5 минут
func formatTime(n int, one, few, many string) string {
	if n%10 == 1 && n%100 != 11 {
		return formatInt(n) + " " + one
	}
	if n%10 >= 2 && n%10 <= 4 && (n%100 < 10 || n%100 >= 20) {
		return formatInt(n) + " " + few
	}
	return formatInt(n) + " " + many
}

// formatInt - конвертирует число в строку
func formatInt(n int) string {
	return string(rune(n + '0'))
}

// GetTimeSince - возвращает время, прошедшее с указанного момента
// Пример: если прошло 3 минуты с startTime, вернет "3m0s"
func GetTimeSince(startTime time.Time) time.Duration {
	return time.Since(startTime)
}

// GetTimeUntil - возвращает время до указанного момента
// Пример: если до endTime осталось 5 минут, вернет "5m0s"
func GetTimeUntil(endTime time.Time) time.Duration {
	return time.Until(endTime)
}

// IsTimeExpired - проверяет, истекло ли время с момента startTime + duration
// Используется для проверки таймаутов
// Пример: startTime = 14:00, duration = 30s, сейчас 14:00:35 -> вернет true
func IsTimeExpired(startTime time.Time, duration time.Duration) bool {
	return time.Since(startTime) > duration
}

// AddDuration - добавляет duration к указанному времени
// Пример: AddDuration(now, 5*time.Minute) вернет время через 5 минут
func AddDuration(t time.Time, d time.Duration) time.Time {
	return t.Add(d)
}

// SubtractDuration - вычитает duration из указанного времени
// Пример: SubtractDuration(now, 5*time.Minute) вернет время 5 минут назад
func SubtractDuration(t time.Time, d time.Duration) time.Time {
	return t.Add(-d)
}

// DifferenceInSeconds - возвращает разницу между двумя временами в секундах
// Всегда возвращает положительное число
func DifferenceInSeconds(t1, t2 time.Time) int64 {
	diff := t1.Sub(t2)
	if diff < 0 {
		diff = -diff
	}
	return int64(diff.Seconds())
}

// IsBefore - проверяет, что t1 раньше чем t2
func IsBefore(t1, t2 time.Time) bool {
	return t1.Before(t2)
}

// IsAfter - проверяет, что t1 позже чем t2
func IsAfter(t1, t2 time.Time) bool {
	return t1.After(t2)
}

// IsEqual - проверяет, что два времени равны
func IsEqual(t1, t2 time.Time) bool {
	return t1.Equal(t2)
}

// Sleep - останавливает выполнение на указанное время
// Простая обертка над time.Sleep для удобства
func Sleep(d time.Duration) {
	time.Sleep(d)
}

// GetCurrentTime - возвращает текущее время
func GetCurrentTime() time.Time {
	return time.Now()
}

// GetCurrentTimeUTC - возвращает текущее время в UTC
func GetCurrentTimeUTC() time.Time {
	return time.Now().UTC()
}

// FormatTime - форматирует время в заданный формат
// Часто используемые форматы:
//   - "2006-01-02 15:04:05" -> "2025-10-28 14:30:00"
//   - "15:04:05" -> "14:30:00"
//   - "2006-01-02" -> "2025-10-28"
func FormatTime(t time.Time, layout string) string {
	return t.Format(layout)
}

// FormatTimeForLog - форматирует время для логов (короткий формат)
// Формат: "14:30:05"
func FormatTimeForLog(t time.Time) string {
	return t.Format("15:04:05")
}

// FormatTimeForDisplay - форматирует время для отображения (полный формат)
// Формат: "2025-10-28 14:30:05"
func FormatTimeForDisplay(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// GetDayStart - возвращает начало текущего дня (00:00:00)
func GetDayStart() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

// GetDayEnd - возвращает конец текущего дня (23:59:59)
func GetDayEnd() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
}

// IsToday - проверяет, что указанное время - сегодня
func IsToday(t time.Time) bool {
	now := time.Now()
	return t.Year() == now.Year() && t.Month() == now.Month() && t.Day() == now.Day()
}

// IsYesterday - проверяет, что указанное время - вчера
func IsYesterday(t time.Time) bool {
	yesterday := time.Now().AddDate(0, 0, -1)
	return t.Year() == yesterday.Year() && t.Month() == yesterday.Month() && t.Day() == yesterday.Day()
}
