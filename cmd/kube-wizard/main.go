package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/SzymonSkrzypczyk/k8s-wizard/internal/app"
	"github.com/SzymonSkrzypczyk/k8s-wizard/internal/logger"
)

var version = "0.1.0"

func getDetailedVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return version
	}

	// Use ldflags version if it matches the default/placeholder
	// or if we want to prioritize it.
	if version != "0.1.0" && version != "" {
		return version
	}

	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}

	return version
}

func printUsage() {
	fmt.Println("kube-wizard - interactive kubectl command wizard")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  kube-wizard [--version] [--config PATH]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -h, --help       Show this help message and exit")
	fmt.Println("      --version    Print the version and exit")
	fmt.Println("      --config     Path to optional configuration file (not yet used)")
}

func main() {
	// Initialize logger
	logPath, err := logger.Init()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to initialize logger: %v\n", err)
	} else {
		defer logger.Close()
		// We could print the log path if we wanted to be helpful, 
		// but let's keep it quiet for now unless it's needed.
		_ = logPath
	}

	// Minimal hand-rolled flag parsing to keep behaviour explicit and avoid
	// starting the TUI when the user only wants help or version information.
	args := os.Args[1:]
	showHelp := false
	showVersion := false
	configPath := ""

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			showHelp = true
		case arg == "--version":
			showVersion = true
		case arg == "--config":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: --config flag requires a path argument")
				fmt.Fprintln(os.Stderr)
				printUsage()
				os.Exit(2)
			}
			configPath = args[i+1]
			i++
		case strings.HasPrefix(arg, "--config="):
			configPath = strings.TrimPrefix(arg, "--config=")
		default:
			fmt.Fprintf(os.Stderr, "Error: unknown flag or argument %q\n\n", arg)
			printUsage()
			os.Exit(2)
		}
	}

	if showHelp {
		printUsage()
		return
	}

	if showVersion {
		fmt.Printf("kube-wizard version %s\n", getDetailedVersion())
		return
	}

	// For now, the config path is parsed but not yet wired into the app.
	_ = configPath

	// Check if kubectl is installed
	kubectlClient := app.NewModel().GetKubectlClient()
	if err := kubectlClient.CheckKubectlInstalled(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Check kubectl version compatibility
	major, minor, err := kubectlClient.GetKubectlVersion()
	if err == nil {
		if major < 1 || (major == 1 && minor < 21) {
			fmt.Fprintf(os.Stderr, "Warning: kubectl version %d.%d is older than the recommended v1.21+\n", major, minor)
			fmt.Fprintln(os.Stderr, "Some features may not work as expected.")
			fmt.Fprintln(os.Stderr)
		}
	}

	// Initialize the Bubble Tea program with our app model
	p := tea.NewProgram(
		app.NewModel(),
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running application: %v\n", err)
		os.Exit(1)
	}
}
