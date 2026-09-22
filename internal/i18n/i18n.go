package i18n

import (
	"embed"
	"fmt"
	"os"
	"sort"
	"strings"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pelletier/go-toml/v2"
	"golang.org/x/text/language"
)

type MessageID string
type Language struct{ Tag, Name string }

const (
	Title                MessageID = "app.title"
	LanguageEnglish      MessageID = "language.english"
	LanguageRussian      MessageID = "language.russian"
	Learn                MessageID = "menu.learn"
	Improve              MessageID = "menu.improve"
	CustomText           MessageID = "menu.text"
	History              MessageID = "menu.history"
	Settings             MessageID = "menu.settings"
	Help                 MessageID = "menu.help"
	Quit                 MessageID = "menu.quit"
	LearnDesc            MessageID = "menu.learn_desc"
	ImproveDesc          MessageID = "menu.improve_desc"
	TextDesc             MessageID = "menu.text_desc"
	HistoryDesc          MessageID = "menu.history_desc"
	SettingsDesc         MessageID = "menu.settings_desc"
	HelpDesc             MessageID = "menu.help_desc"
	QuitDesc             MessageID = "menu.quit_desc"
	Paused               MessageID = "practice.paused"
	PauseContinue        MessageID = "practice.pause_continue"
	PauseRestart         MessageID = "practice.pause_restart"
	PauseExit            MessageID = "practice.pause_exit"
	PasteDisabled        MessageID = "practice.paste_disabled"
	PracticeHeader       MessageID = "practice.header"
	FingerHint           MessageID = "practice.finger"
	SpaceFingerHint      MessageID = "practice.space_finger"
	LearnProgress        MessageID = "practice.learn_progress"
	LearnProgressHelp    MessageID = "practice.learn_progress_help"
	ResultTitle          MessageID = "result.title"
	ResultTarget         MessageID = "result.target"
	ResultStats          MessageID = "result.stats"
	TextEmpty            MessageID = "text.empty"
	TextChoose           MessageID = "text.choose"
	InvalidUTF8          MessageID = "error.invalid_utf8"
	FileError            MessageID = "error.file"
	StoreError           MessageID = "error.store"
	HistorySessions      MessageID = "history.sessions"
	HistoryKeys          MessageID = "history.keys"
	HistoryTrends        MessageID = "history.trends"
	NoHistory            MessageID = "history.empty"
	ColDate              MessageID = "column.date"
	ColMode              MessageID = "column.mode"
	ColLang              MessageID = "column.language"
	ColWPM               MessageID = "column.wpm"
	ColAccuracy          MessageID = "column.accuracy"
	ColErrors            MessageID = "column.errors"
	ColKey               MessageID = "column.key"
	ColSamples           MessageID = "column.samples"
	ColLatency           MessageID = "column.latency"
	ColConfidence        MessageID = "column.confidence"
	ModeLearn            MessageID = "mode.learn"
	ModeImprove          MessageID = "mode.improve"
	ModeText             MessageID = "mode.text"
	Calibration          MessageID = "mode.calibration"
	SettingsUI           MessageID = "settings.ui_language"
	SettingsTraining     MessageID = "settings.training_language"
	SettingsSpeed        MessageID = "settings.speed"
	SettingsAccuracy     MessageID = "settings.accuracy"
	SettingsLength       MessageID = "settings.length"
	SettingsKeyboard     MessageID = "settings.keyboard"
	SettingsTheme        MessageID = "settings.theme"
	SettingsPosition     MessageID = "settings.position"
	ThemeViolet          MessageID = "theme.violet"
	ThemeOcean           MessageID = "theme.ocean"
	ThemeSunset          MessageID = "theme.sunset"
	ThemeMono            MessageID = "theme.mono"
	PositionTopLeft      MessageID = "position.top_left"
	PositionTopCenter    MessageID = "position.top_center"
	PositionTopRight     MessageID = "position.top_right"
	PositionCenterLeft   MessageID = "position.center_left"
	PositionCenter       MessageID = "position.center"
	PositionCenterRight  MessageID = "position.center_right"
	PositionBottomLeft   MessageID = "position.bottom_left"
	PositionBottomCenter MessageID = "position.bottom_center"
	PositionBottomRight  MessageID = "position.bottom_right"
	SettingsReset        MessageID = "settings.reset"
	SettingsConfirm      MessageID = "settings.confirm_reset"
	SettingsResetDone    MessageID = "settings.reset_done"
	BoolYes              MessageID = "boolean.yes"
	BoolNo               MessageID = "boolean.no"
	HelpBody             MessageID = "help.body"
	HelpFingers          MessageID = "help.fingers"
	FingerLP             MessageID = "finger.lp"
	FingerLR             MessageID = "finger.lr"
	FingerLM             MessageID = "finger.lm"
	FingerLI             MessageID = "finger.li"
	FingerRI             MessageID = "finger.ri"
	FingerRM             MessageID = "finger.rm"
	FingerRR             MessageID = "finger.rr"
	FingerRP             MessageID = "finger.rp"
	HotkeyQuit           MessageID = "hotkey.quit"
	HotkeyBack           MessageID = "hotkey.back"
	HotkeyPause          MessageID = "hotkey.pause"
	HotkeyPractice       MessageID = "hotkey.practice"
	HotkeyResult         MessageID = "hotkey.result"
	HotkeyText           MessageID = "hotkey.text"
	HotkeyHistory        MessageID = "hotkey.history"
	HotkeySettings       MessageID = "hotkey.settings"
	TerminalSmall        MessageID = "terminal.small"
	CLILang              MessageID = "cli.lang"
	CLIMode              MessageID = "cli.mode"
	CLIText              MessageID = "cli.text"
	CLIDataDir           MessageID = "cli.data_dir"
	CLIVersion           MessageID = "cli.version"
	CLIInvalidLang       MessageID = "cli.invalid_lang"
	CLIInvalidMode       MessageID = "cli.invalid_mode"
	CLIError             MessageID = "cli.error"
	CountLessons         MessageID = "count.lessons"
	CountErrors          MessageID = "count.errors"
	CountCharacters      MessageID = "count.characters"
	CountSamples         MessageID = "count.samples"
)

//go:embed active.*.toml
var catalogs embed.FS

type Localizer struct {
	bundle    *goi18n.Bundle
	localizer *goi18n.Localizer
	lang      string
}

func New(tag string) (*Localizer, error) {
	b := goi18n.NewBundle(language.English)
	b.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	for _, name := range []string{"active.en.toml", "active.ru.toml"} {
		if _, err := b.LoadMessageFileFS(catalogs, name); err != nil {
			return nil, fmt.Errorf("load translation catalog %s: %w", name, err)
		}
	}
	if err := validateCatalogs(); err != nil {
		return nil, err
	}
	l := &Localizer{bundle: b}
	_ = l.SetLanguage(tag)
	return l, nil
}
func normalize(tag string) string {
	tag = strings.TrimSpace(strings.Split(strings.Split(tag, ".")[0], "@")[0])
	tag = strings.ReplaceAll(tag, "_", "-")
	parsed, err := language.Parse(tag)
	if err != nil {
		return "en"
	}
	base, _ := parsed.Base()
	if base.String() == "ru" {
		return "ru"
	}
	return "en"
}
func Detect() string {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := os.Getenv(key); v != "" {
			return normalize(v)
		}
	}
	return "en"
}
func (l *Localizer) SetLanguage(tag string) error {
	l.lang = normalize(tag)
	l.localizer = goi18n.NewLocalizer(l.bundle, l.lang, "en")
	return nil
}
func (l *Localizer) Language() string                              { return l.lang }
func (l *Localizer) Text(id MessageID, data map[string]any) string { return l.localize(id, nil, data) }
func (l *Localizer) Plural(id MessageID, count any, data map[string]any) string {
	return l.localize(id, count, data)
}
func (l *Localizer) localize(id MessageID, count any, data map[string]any) string {
	s, err := l.localizer.Localize(&goi18n.LocalizeConfig{MessageID: string(id), PluralCount: count, TemplateData: data})
	if err != nil {
		return "!" + string(id) + "!"
	}
	return s
}
func (l *Localizer) SupportedLanguages() []Language {
	return []Language{{"en", l.Text(LanguageEnglish, nil)}, {"ru", l.Text(LanguageRussian, nil)}}
}
func validateCatalogs() error {
	read := func(name string) (map[string]any, error) {
		b, e := catalogs.ReadFile(name)
		if e != nil {
			return nil, e
		}
		var v map[string]any
		e = toml.Unmarshal(b, &v)
		return v, e
	}
	en, e := read("active.en.toml")
	if e != nil {
		return e
	}
	ru, e := read("active.ru.toml")
	if e != nil {
		return e
	}
	flatten := func(root map[string]any) map[string]bool {
		out := map[string]bool{}
		var walk func(string, map[string]any)
		walk = func(prefix string, values map[string]any) {
			for key, value := range values {
				path := key
				if prefix != "" {
					path = prefix + "." + key
				}
				if child, ok := value.(map[string]any); ok {
					walk(path, child)
				} else if key == "other" || key == "one" || key == "few" || key == "many" {
					out[prefix] = true
				}
			}
		}
		walk("", root)
		return out
	}
	var missing []string
	for id := range flatten(en) {
		if !flatten(ru)[id] {
			missing = append(missing, id)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		return fmt.Errorf("Russian translation catalog is missing: %s", strings.Join(missing, ", "))
	}
	return nil
}
