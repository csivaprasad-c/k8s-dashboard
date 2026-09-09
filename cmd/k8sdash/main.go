// Command k8sdash is a terminal dashboard for Kubernetes clusters: pick a
// context from your kubeconfig, connect, and browse the cluster live.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/csivaprasad/k8s-dashboard/internal/ui"
)

// version is set at build time via -ldflags "-X main.version=...";
// see the Makefile.
var version = "dev"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		fmt.Println("k8sdash " + version)
		return
	}

	p := tea.NewProgram(ui.New(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "k8sdash: "+err.Error())
		os.Exit(1)
	}
}
