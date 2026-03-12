package tests

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"unicode"
)

var (
	reManyQuestionMarks = regexp.MustCompile(`\?{3,}`)
	reHasCyrillic       = regexp.MustCompile(`[\p{Cyrillic}]`)
	reHasLatin          = regexp.MustCompile(`[A-Za-z]`)
	reMojibakeMarkers   = regexp.MustCompile(`(?:Гђ|Г‘|Гўв‚¬вЂќ|Гўв‚¬вЂњ|Гўв‚¬|ГўвЂћ|РІР‚|Р [A-Za-z]|РЎ[A-Za-z])`)
)

func TestI18N_NoArtifacts(t *testing.T) {
	ru := mustLoadLang(t, filepath.Join("..", "gui", "static", "i18n", "ru.json"))
	en := mustLoadLang(t, filepath.Join("..", "gui", "static", "i18n", "en.json"))

	var issues []string
	check := func(lang, key, value string) {
		if strings.ContainsRune(value, unicode.ReplacementChar) || strings.Contains(value, "\uFFFD") {
			issues = append(issues, lang+":"+key+": contains replacement char")
		}
		if hasSuspiciousControlChars(value) {
			issues = append(issues, lang+":"+key+": contains control chars")
		}
		if reManyQuestionMarks.MatchString(value) {
			issues = append(issues, lang+":"+key+": contains ??? sequence")
		}
		if lang == "ru" && containsForbiddenRuCyrillic(value) {
			issues = append(issues, lang+":"+key+": contains suspicious Cyrillic letters")
		}
		if reMojibakeMarkers.MatchString(value) {
			issues = append(issues, lang+":"+key+": contains mojibake marker sequence")
		}
		if lang == "en" && reHasCyrillic.MatchString(value) {
			issues = append(issues, lang+":"+key+": contains Cyrillic characters")
		}
	}

	for k, v := range ru {
		check("ru", k, v)
	}
	for k, v := range en {
		check("en", k, v)
	}

	if len(issues) > 0 {
		sort.Strings(issues)
		t.Fatalf("i18n artifacts found: %v", sample(issues))
	}
}

func TestI18N_NoNewUntranslatedPhrases(t *testing.T) {
	ru := mustLoadLang(t, filepath.Join("..", "gui", "static", "i18n", "ru.json"))
	en := mustLoadLang(t, filepath.Join("..", "gui", "static", "i18n", "en.json"))

	var newIssues []string
	for k, v := range ru {
		if looksUntranslatedForRU(v) {
			newIssues = append(newIssues, "ru:"+k)
		}
	}
	for k, v := range en {
		if looksUntranslatedForEN(v) {
			newIssues = append(newIssues, "en:"+k)
		}
	}

	if len(newIssues) > 0 {
		sort.Strings(newIssues)
		t.Fatalf("new untranslated i18n phrases detected: %v", sample(newIssues))
	}
}

func looksUntranslatedForRU(value string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}
	if reHasCyrillic.MatchString(value) {
		return false
	}
	if !reHasLatin.MatchString(value) {
		return false
	}
	clean := stripI18NPlaceholders(value)
	clean = strings.TrimSpace(clean)
	return strings.Contains(clean, " ")
}

func looksUntranslatedForEN(value string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}
	return reHasCyrillic.MatchString(value)
}

func stripI18NPlaceholders(value string) string {
	out := reNamedPlaceholder.ReplaceAllString(value, "")
	out = rePrintfPlaceholder.ReplaceAllString(out, "")
	return out
}

func hasSuspiciousControlChars(s string) bool {
	for _, r := range s {
		if !unicode.IsControl(r) {
			continue
		}
		if r == '\n' || r == '\r' || r == '\t' {
			continue
		}
		return true
	}
	return false
}

func containsForbiddenRuCyrillic(s string) bool {
	forbidden := map[rune]struct{}{
		'Ђ': {}, 'ђ': {},
		'Ѓ': {}, 'ѓ': {},
		'Є': {}, 'є': {},
		'І': {}, 'і': {},
		'Ї': {}, 'ї': {},
		'Ј': {}, 'ј': {},
		'Љ': {}, 'љ': {},
		'Њ': {}, 'њ': {},
		'Ћ': {}, 'ћ': {},
		'Џ': {}, 'џ': {},
	}
	for _, r := range s {
		if _, ok := forbidden[r]; ok {
			return true
		}
	}
	return false
}
