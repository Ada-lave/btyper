package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"btyper/internal/application"
	"btyper/internal/domain"
	"btyper/internal/i18n"
	"btyper/internal/storage"
	"btyper/internal/trainer"
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
	if len(os.Args) > 1 && (os.Args[1] == "export" || os.Args[1] == "import") {
		if err := runDataCommand(os.Args[1], os.Args[2:]); err != nil {
			fatal(loc, err)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "profile" {
		if err := runProfileCommand(os.Args[2:]); err != nil {
			fatal(loc, err)
		}
		return
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
		profiles := trainer.Profiles()
		userProfiles, err := store.UserProfiles()
		if err != nil {
			fatal(loc, err)
		}
		for id, profile := range userProfiles {
			profiles[id] = profile
		}
		if _, ok := profiles[*lang]; !ok {
			fatal(loc, fmt.Errorf("%s", loc.Text(i18n.CLIInvalidLang, map[string]any{"Value": fmt.Sprintf("%q", *lang)})))
		}
		settings.Language = *lang
	}
	if *mode != "" {
		m := domain.Mode(*mode)
		if m == domain.ModeLearn || m == domain.ModeImprove {
			m = domain.ModeAdaptive
		}
		if m != domain.ModeAdaptive && m != domain.ModeText {
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
		m := domain.Mode(*mode)
		if m == domain.ModeLearn || m == domain.ModeImprove {
			m = domain.ModeAdaptive
		}
		model.Start(m)
	}
	if _, err := tea.NewProgram(model).Run(); err != nil {
		fatal(loc, err)
	}
}

func runProfileCommand(args []string) error {
	if len(args) == 0 {
		return errors.New("profile command requires import, export, or list")
	}
	command := args[0]
	fs := flag.NewFlagSet("btyper profile "+command, flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "data directory")
	input := fs.String("input", "", "profile JSON input file")
	output := fs.String("output", "", "profile JSON output file")
	id := fs.String("id", "", "profile ID")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	store, err := storage.Open(*dataDir)
	if err != nil {
		return err
	}
	defer store.Close()
	switch command {
	case "import":
		if *input == "" {
			return errors.New("--input is required")
		}
		file, err := os.Open(*input)
		if err != nil {
			return err
		}
		defer file.Close()
		profile, err := store.ImportUserProfile(file)
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, "imported profile:", profile.ID)
	case "export":
		if *id == "" || *output == "" {
			return errors.New("--id and --output are required")
		}
		var data bytes.Buffer
		if err := store.ExportUserProfile(*id, &data); err != nil {
			return err
		}
		file, err := os.OpenFile(*output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		if _, err = file.Write(data.Bytes()); err != nil {
			file.Close()
			return err
		}
		return file.Close()
	case "list":
		profiles := trainer.Profiles()
		users, err := store.UserProfiles()
		if err != nil {
			return err
		}
		for key, profile := range users {
			profiles[key] = profile
		}
		var ids []string
		for key := range profiles {
			ids = append(ids, key)
		}
		sort.Strings(ids)
		for _, key := range ids {
			fmt.Fprintln(os.Stdout, key)
		}
	default:
		return fmt.Errorf("unknown profile command %q", command)
	}
	return nil
}

func runDataCommand(command string, args []string) error {
	fs := flag.NewFlagSet("btyper "+command, flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "data directory")
	input := fs.String("input", "", "backup input file")
	output := fs.String("output", "", "backup output file")
	outputDir := fs.String("output-dir", "", "CSV output directory")
	format := fs.String("format", "backup", "backup or csv")
	if err := fs.Parse(args); err != nil {
		return err
	}
	store, err := storage.Open(*dataDir)
	if err != nil {
		return err
	}
	defer store.Close()
	if command == "export" {
		switch *format {
		case "backup":
			if *output == "" {
				return errors.New("--output is required for backup export")
			}
			f, err := os.OpenFile(*output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
			if err != nil {
				return err
			}
			if err = store.ExportBackup(f); err != nil {
				f.Close()
				return err
			}
			return f.Close()
		case "csv":
			if *outputDir == "" {
				return errors.New("--output-dir is required for CSV export")
			}
			return store.ExportCSV(*outputDir)
		default:
			return fmt.Errorf("unknown export format %q", *format)
		}
	}
	if *input == "" {
		return errors.New("--input is required")
	}
	backupPath := filepath.Join(store.DataDir(), "pre-import-"+time.Now().Format("20060102T150405.000000000")+".json")
	backup, err := os.OpenFile(backupPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if err = store.ExportBackup(backup); err != nil {
		backup.Close()
		return err
	}
	if err = backup.Close(); err != nil {
		return err
	}
	f, err := os.Open(*input)
	if err != nil {
		return err
	}
	defer f.Close()
	if err = store.ImportBackup(f); err != nil {
		return fmt.Errorf("import failed (current data preserved; safety backup: %s): %w", backupPath, err)
	}
	fmt.Fprintln(os.Stdout, "safety backup:", backupPath)
	return nil
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
