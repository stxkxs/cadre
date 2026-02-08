package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	logsAgent  string
	logsFollow bool
	logsLines  int
	logsRunID  string
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View execution logs",
	Long: `View logs from crew execution.

Examples:
  cadre logs                    # View recent logs
  cadre logs --agent developer  # Filter by agent
  cadre logs --follow           # Follow log output
  cadre logs --run abc123       # Logs for specific run`,
	RunE: runLogs,
}

func init() {
	logsCmd.Flags().StringVarP(&logsAgent, "agent", "a", "", "filter logs by agent")
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "follow log output")
	logsCmd.Flags().IntVarP(&logsLines, "lines", "n", 50, "number of lines to show")
	logsCmd.Flags().StringVar(&logsRunID, "run", "", "show logs for specific run ID")
}

func runLogs(cmd *cobra.Command, args []string) error {
	logsDir := ".cadre/logs"

	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		fmt.Println("No logs found.")
		return nil
	}

	if logsFollow {
		return followLogs(logsDir)
	}

	return showLogs(logsDir)
}

func showLogs(logsDir string) error {
	// Find log files
	var logFiles []string

	err := filepath.Walk(logsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".log") {
			// Filter by run ID if specified
			if logsRunID != "" && !strings.Contains(path, logsRunID) {
				return nil
			}
			// Filter by agent if specified
			if logsAgent != "" && !strings.Contains(path, logsAgent) {
				return nil
			}
			logFiles = append(logFiles, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to scan logs: %w", err)
	}

	if len(logFiles) == 0 {
		fmt.Println("No matching logs found.")
		return nil
	}

	// Read and display logs
	for _, logFile := range logFiles {
		content, err := readLastLines(logFile, logsLines)
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", logFile, err)
			continue
		}

		fmt.Printf("=== %s ===\n", filepath.Base(logFile))
		fmt.Println(content)
		fmt.Println()
	}

	return nil
}

func followLogs(logsDir string) error {
	fmt.Println("Following logs... (Ctrl+C to stop)")

	// Simple tail -f implementation
	mainLog := filepath.Join(logsDir, "cadre.log")

	file, err := os.Open(mainLog)
	if err != nil {
		// If main log doesn't exist, create placeholder
		if os.IsNotExist(err) {
			fmt.Println("Waiting for logs...")
			for {
				time.Sleep(time.Second)
				if _, err := os.Stat(mainLog); err == nil {
					file, _ = os.Open(mainLog)
					break
				}
			}
		} else {
			return fmt.Errorf("failed to open log file: %w", err)
		}
	}
	defer file.Close()

	// Seek to end
	file.Seek(0, 2)

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Filter by agent if specified
		if logsAgent != "" && !strings.Contains(line, logsAgent) {
			continue
		}

		fmt.Print(line)
	}
}

func readLastLines(filepath string, n int) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > n {
			lines = lines[1:]
		}
	}

	return strings.Join(lines, "\n"), scanner.Err()
}
