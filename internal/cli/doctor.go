package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/stxkxs/cadre/internal/config"
	"github.com/stxkxs/cadre/internal/state"
	"github.com/stxkxs/cadre/internal/tool"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check environment and dependencies",
	Long:  "Validate that all required dependencies, configuration, and tools are properly set up.",
	RunE:  runDoctor,
}

func runDoctor(cmd *cobra.Command, args []string) error {
	fmt.Println("cadre doctor — checking your environment")
	fmt.Println()
	allOK := true

	// 1. Go version
	fmt.Printf("  Go version: %s", runtime.Version())
	fmt.Println(" ✓")

	// 2. OS/Arch
	fmt.Printf("  Platform:   %s/%s", runtime.GOOS, runtime.GOARCH)
	fmt.Println(" ✓")

	// 3. API key
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey != "" {
		fmt.Printf("  API key:    set (***%s)", apiKey[max(0, len(apiKey)-4):])
		fmt.Println(" ✓")
	} else {
		fmt.Println("  API key:    NOT SET ✗")
		fmt.Println("    → Set ANTHROPIC_API_KEY environment variable")
		allOK = false
	}

	// 4. Configuration
	cfg, err := config.Load(".")
	if err != nil {
		fmt.Println("  Config:     NOT FOUND ✗")
		fmt.Printf("    → Run 'cadre init' to create a project\n")
		allOK = false
	} else {
		fmt.Printf("  Config:     %s v%s", cfg.Name, cfg.Version)
		fmt.Println(" ✓")
	}

	// 5. State database
	if cfg != nil {
		_, err := state.NewManager(cfg.State.Driver, cfg.State.Path)
		if err != nil {
			fmt.Printf("  State DB:   FAILED (%s) ✗\n", err)
			allOK = false
		} else {
			fmt.Printf("  State DB:   %s (%s)", cfg.State.Driver, cfg.State.Path)
			fmt.Println(" ✓")
		}
	}

	// 6. Built-in tools
	tools := tool.ListBuiltins()
	fmt.Printf("  Tools:      %d built-in", len(tools))
	fmt.Println(" ✓")

	// 7. Git
	if _, err := exec.LookPath("git"); err == nil {
		fmt.Println("  Git:        available ✓")
	} else {
		fmt.Println("  Git:        NOT FOUND ✗")
		allOK = false
	}

	fmt.Println()
	if allOK {
		fmt.Println("All checks passed!")
	} else {
		fmt.Println("Some checks failed. See above for details.")
	}

	return nil
}
