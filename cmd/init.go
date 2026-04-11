package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/anthropics/ferry/internal/config"
	"github.com/anthropics/ferry/internal/registry"
	"github.com/anthropics/ferry/internal/tui"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactively configure profiles and targets",
	RunE:  runInit,
}

func init() {
	initCmd.Flags().String("profile", "", "Edit a specific profile directly (skip profile manager)")
}

func runInit(cmd *cobra.Command, args []string) error {
	profileFlag, _ := cmd.Flags().GetString("profile")

	// Load existing lock file (best effort)
	var lf *config.LockFile
	existing, err := config.ReadLockFile()
	if err == nil {
		lf = existing
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("reading ferry.lock: %w", err)
	}
	if lf == nil {
		lf = config.DefaultLockFile()
	}

	// --profile flag: jump directly to wizard for that profile
	if profileFlag != "" {
		return runProfileWizard(lf, profileFlag)
	}

	// Profile manager loop
	for {
		result, err := tui.RunProfileManager(lf, false)
		if err != nil {
			return err
		}
		switch result.Action {
		case tui.PMActionQuit:
			return nil

		case tui.PMActionNew:
			// Preset picker
			presetName, aborted, err := tui.RunPresetPicker()
			if err != nil {
				return err
			}
			if aborted {
				continue
			}

			// Name the profile
			name, err := promptProfileName(lf.Profiles)
			if err != nil {
				return err
			}
			if name == "" {
				continue
			}

			// Start from preset or blank
			var base *config.ProfileConfig
			if presetName != "" {
				for _, p := range registry.Presets() {
					if p.Name == presetName {
						base = &p.Profile
						break
					}
				}
			}
			if err := runProfileWizardWithBase(lf, name, base); err != nil {
				return err
			}

		case tui.PMActionEdit:
			if err := runProfileWizard(lf, result.ProfileName); err != nil {
				return err
			}

		case tui.PMActionDelete:
			confirmed, err := tui.ConfirmPrompt(
				fmt.Sprintf("Delete profile %q from ferry.lock?", result.ProfileName))
			if err != nil {
				return err
			}
			if confirmed {
				delete(lf.Profiles, result.ProfileName)
				if err := config.WriteLockFile(lf); err != nil {
					return err
				}
				fmt.Printf("  deleted profile %q\n", result.ProfileName)
			}

		case tui.PMActionBuild:
			fmt.Printf("  run: ferry bundle --profile %s\n", result.ProfileName)
			return nil
		}
	}
}

func runProfileWizard(lf *config.LockFile, profileName string) error {
	existing := lf.Profiles[profileName]
	return runProfileWizardWithBase(lf, profileName, &existing)
}

func runProfileWizardWithBase(lf *config.LockFile, profileName string, base *config.ProfileConfig) error {
	prof, aborted, err := tui.RunProfileWizard(profileName, base)
	if err != nil {
		return err
	}
	if aborted {
		return nil
	}
	if lf.Profiles == nil {
		lf.Profiles = make(map[string]config.ProfileConfig)
	}
	lf.Profiles[profileName] = *prof
	if err := config.WriteLockFile(lf); err != nil {
		return err
	}
	fmt.Printf("✓ wrote ferry.lock (profile: %s)\n", profileName)
	fmt.Printf("  run: ferry bundle --profile %s\n", profileName)
	return nil
}

func promptProfileName(existing map[string]config.ProfileConfig) (string, error) {
	fmt.Print("  Profile name: ")
	var name string
	_, err := fmt.Scanln(&name)
	if err != nil || name == "" {
		return "", nil
	}
	if _, exists := existing[name]; exists {
		fmt.Printf("  profile %q already exists — editing instead\n", name)
	}
	return name, nil
}
