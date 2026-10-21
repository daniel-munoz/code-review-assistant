package cmd

import (
	"fmt"
	"os"

	"github.com/daniel-munoz/code-review-assistant/internal/config"
	"github.com/spf13/cobra"
)

// listPresetsCmd represents the list-presets command
var listPresetsCmd = &cobra.Command{
	Use:   "list-presets",
	Short: "List the built-in analysis presets",
	Long: `List the built-in analysis presets with a short description of each.

Select a preset with "analyze --preset=<name>", the CRA_PRESET environment
variable, or "preset: <name>" in a config file. Use show-preset to see the
values a preset sets.`,
	Args: cobra.NoArgs,
	RunE: runListPresets,
}

// showPresetCmd represents the show-preset command
var showPresetCmd = &cobra.Command{
	Use:   "show-preset <name|path>",
	Short: "Show the values an analysis preset sets",
	Long: `Print a preset's YAML. For a built-in preset this is the embedded file,
which you can copy as a starting point for a custom preset.

Example usage:
  code-review-assistant show-preset strict
  code-review-assistant show-preset strict > team-preset.yaml
  code-review-assistant show-preset ./team-preset.yaml   # validate a custom preset`,
	Args: cobra.ExactArgs(1),
	RunE: runShowPreset,
}

func init() {
	rootCmd.AddCommand(listPresetsCmd)
	rootCmd.AddCommand(showPresetCmd)
}

func runListPresets(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()
	for _, p := range config.BuiltinPresets() {
		fmt.Fprintf(out, "%-10s %s\n", p.Name, p.Description)
	}
	return nil
}

func runShowPreset(cmd *cobra.Command, args []string) error {
	value := args[0]

	var data []byte
	if config.IsPresetPath(value) {
		if _, err := config.LoadPreset(value, ""); err != nil {
			return err
		}
		var err error
		if data, err = os.ReadFile(value); err != nil {
			return fmt.Errorf("failed to read preset %q: %w", value, err)
		}
	} else {
		p, err := config.LoadPreset(value, "")
		if err != nil {
			return err
		}
		if data, err = config.BuiltinPresetYAML(p.Label); err != nil {
			return err
		}
	}

	_, err := cmd.OutOrStdout().Write(data)
	return err
}
