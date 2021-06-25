package combatlog

import "regexp"

//go:generate protoc -I ../protobuf/ --go_opt=module=github.com/shivas/abyss-blackbox --go_out=.. combatlog.proto

// LanguageMatchers holds default initialized regexps for all languages.
var LanguageMatchers LocalizedMatchers

func init() {
	LanguageMatchers = make(LocalizedMatchers)

	LanguageMatchers[LanguageCode_ENGLISH] = LocalizedMatcher{
		ListenerRe:     regexp.MustCompile(`(?m)^\s*Listener:\s(.*)$`),
		SessionStartRe: regexp.MustCompile(`(?m)^\s*Session Started:\s(.*)$`),
	}

	LanguageMatchers[LanguageCode_FRENCH] = LocalizedMatcher{
		ListenerRe:     regexp.MustCompile(`(?m)^\s*Auditeur:\s(.*)$`),
		SessionStartRe: regexp.MustCompile(`(?m)^\s*Session commencée:\s(.*)$`),
	}

	LanguageMatchers[LanguageCode_GERMAN] = LocalizedMatcher{
		ListenerRe:     regexp.MustCompile(`(?m)^\s*Empfänger:\s(.*)$`),
		SessionStartRe: regexp.MustCompile(`(?m)^\s*Sitzung gestartet:\s(.*)$`),
	}

	LanguageMatchers[LanguageCode_RUSSIAN] = LocalizedMatcher{
		ListenerRe:     regexp.MustCompile(`(?m)^\s*Слушатель:\s(.*)$`),
		SessionStartRe: regexp.MustCompile(`(?m)^\s*Сеанс начат:\s(.*)$`),
	}

	LanguageMatchers[LanguageCode_JAPANESE] = LocalizedMatcher{
		ListenerRe:     regexp.MustCompile(`(?m)^\s*傍聴者:\s(.*)$`),
		SessionStartRe: regexp.MustCompile(`(?m)^\s*セッション開始:\s(.*)$`),
	}

	LanguageMatchers[LanguageCode_KOREAN] = LocalizedMatcher{
		ListenerRe:     regexp.MustCompile(`(?m)^\s*청취자:\s(.*)$`),
		SessionStartRe: regexp.MustCompile(`(?m)^\s*세션 시작됨:\s(.*)$`),
	}

	LanguageMatchers[LanguageCode_CHINESE] = LocalizedMatcher{
		ListenerRe:     regexp.MustCompile(`(?m)^\s*收听者:\s(.*)$`),
		SessionStartRe: regexp.MustCompile(`(?m)^\s*进程开始:\s(.*)$`),
	}
}

// LocalizedMatcher holds regexps for language to detect.
type LocalizedMatcher struct {
	ListenerRe     *regexp.Regexp
	SessionStartRe *regexp.Regexp
}

// LocalizedMatchers mapping between LanguageCode and LocalizedMatchers.
type LocalizedMatchers map[LanguageCode]LocalizedMatcher
