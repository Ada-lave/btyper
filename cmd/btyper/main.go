package main

import (
	"flag"
	"fmt"
	"os"

	"btyper/internal/application"
	"btyper/internal/domain"
	"btyper/internal/i18n"
	"btyper/internal/storage"
	"btyper/internal/ui"
	tea "charm.land/bubbletea/v2"
)

var version = "dev"

func main() {
	loc, err := i18n.New(i18n.Detect())
	if err != nil {
		fmt.Fprintln(os.Stderr, "btyper:", err)
		os.Exit(1)
	}
	lang := flag.String("lang", "", loc.Text(i18n.CLILang, nil))
	mode := flag.String("mode", "", loc.Text(i18n.CLIMode, nil))
	textPath := flag.String("text", "", loc.Text(i18n.CLIText, nil))
	dataDir := flag.String("data-dir", "", loc.Text(i18n.CLIDataDir, nil))
	showVersion := flag.Bool("version", false, loc.Text(i18n.CLIVersion, nil))
	flag.Parse()
	if *showVersion {
		fmt.Println("btyper", version)
		return
	}
	store, err := storage.Open(*dataDir)
	if err != nil {
		fatal(loc, err)
	}
	defer store.Close()
	settings, err := application.LoadSettings(store)
	if err != nil {
		fatal(loc, err)
	}
	settings = applyLocaleDefaults(settings, i18n.Detect(), *lang)
	_ = loc.SetLanguage(settings.UILanguage)
	if *lang != "" {
		if *lang != "en" && *lang != "ru" {
			fatal(loc, fmt.Errorf("%s", loc.Text(i18n.CLIInvalidLang, map[string]any{"Value": fmt.Sprintf("%q", *lang)})))
		}
		settings.Language = *lang
	}
	if *mode != "" {
		m := domain.Mode(*mode)
		if m != domain.ModeLearn && m != domain.ModeImprove && m != domain.ModeText {
			fatal(loc, fmt.Errorf("%s", loc.Text(i18n.CLIInvalidMode, map[string]any{"Value": fmt.Sprintf("%q", *mode)})))
		}
		settings.Mode = m
	}
	var initialText string
	if *textPath != "" {
		initialText, err = application.ReadCustomText(*textPath)
		if err != nil {
			fatal(loc, fmt.Errorf("%s", loc.Text(i18n.FileError, map[string]any{"Error": err})))
		}
		settings.Mode = domain.ModeText
	}
	model, err := ui.New(store, settings)
	if err != nil {
		fatal(loc, err)
	}
	if initialText != "" {
		if err := model.StartCustomText(initialText); err != nil {
			fatal(loc, err)
		}
	} else if *mode != "" {
		model.Start(domain.Mode(*mode))
	}
	if _, err := tea.NewProgram(model).Run(); err != nil {
		fatal(loc, err)
	}
}

func applyLocaleDefaults(settings domain.Settings, detected, cliLanguage string) domain.Settings {
	if settings.UILanguage == "" {
		settings.UILanguage = detected
		if cliLanguage == "" {
			settings.Language = detected
		}
	}
	return settings
}

func fatal(loc *i18n.Localizer, err error) {
	fmt.Fprintln(os.Stderr, loc.Text(i18n.CLIError, map[string]any{"Error": err}))
	os.Exit(1)
}
