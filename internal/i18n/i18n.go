// Package i18n — локалізація UI (uk — джерело, en — переклад).
package i18n

import (
	"strings"
	"sync"
)

var mu sync.RWMutex
var current = "uk"

// Languages — доступні мови.
var Languages = map[string]string{
	"uk": "Українська",
	"en": "English",
}

// SetLanguage — встановлює активну мову, повертає нормалізовану.
func SetLanguage(lang string) string {
	mu.Lock()
	defer mu.Unlock()
	l := strings.ToLower(strings.TrimSpace(lang))
	if _, ok := Languages[l]; ok {
		current = l
	}
	return current
}

// GetLanguage — поточна мова.
func GetLanguage() string {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// Tr — перекладає рядок. Невідомі рядки повертаються як є.
// Плейсхолдери {name} замінюються через kwargs.
func Tr(text string, kwargs ...map[string]string) string {
	if text == "" {
		return text
	}
	mu.RLock()
	lang := current
	mu.RUnlock()

	out := text
	if lang == "en" {
		if en, ok := enMap[text]; ok {
			out = en
		}
	}
	if len(kwargs) > 0 {
		for k, v := range kwargs[0] {
			out = strings.ReplaceAll(out, "{"+k+"}", v)
		}
	}
	return out
}

// Trf — аналогічний Tr з одразу переданими kwargs.
func Trf(template string, kwargs map[string]string) string {
	return Tr(template, kwargs)
}
