package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewUpdateCommand creates the update command
func NewUpdateCommand(container *CLIContainer) *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update Kilometers CLI to the latest version",
		Long: `Update the Kilometers CLI to the latest version.

This command will check for updates and provide instructions
on how to update the CLI tool.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(container)
		},
	}
}

// runUpdate handles the update process
func runUpdate(container *CLIContainer) error {
	fmt.Printf("🔄 Kilometers CLI Update (Current: %s)\n", Version)
	fmt.Println("")

	// For now, provide manual update instructions
	// In a production version, this could check for updates automatically
	fmt.Println("Update functionality coming soon!")
	fmt.Println("")
	fmt.Println("To update to the latest version:")
	fmt.Println("")
	fmt.Println("1. Download the latest version from:")
	fmt.Println("   https://github.com/kilometers-ai/kilometers-cli/releases/latest")
	fmt.Println("")
	fmt.Println("2. Replace your current installation with the new version")
	fmt.Println("")
	fmt.Println("3. Run 'km --version' to verify the update")
	fmt.Println("")
	fmt.Println("Alternative installation methods:")
	fmt.Println("")
	fmt.Println("• Using curl (Linux/macOS):")
	fmt.Println("  curl -fsSL https://install.kilometers.ai/cli | sh")
	fmt.Println("")
	fmt.Println("• Using PowerShell (Windows):")
	fmt.Println("  iwr https://install.kilometers.ai/cli.ps1 | iex")
	fmt.Println("")
	fmt.Println("• Using Homebrew (macOS):")
	fmt.Println("  brew update && brew upgrade kilometers-cli")
	fmt.Println("")
	fmt.Println("• Using npm:")
	fmt.Println("  npm update -g @kilometers/cli")
	fmt.Println("")
	fmt.Printf("Current version: %s\n", Version)
	fmt.Printf("Build time: %s\n", BuildTime)
	fmt.Println("")
	fmt.Println("📝 Note: Your configuration will be preserved during the update")

	return nil
}
