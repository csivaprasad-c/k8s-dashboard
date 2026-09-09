// Package ui wires the three screens (cluster select, connecting, dashboard)
// together behind a single root Bubble Tea model.
package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/csivaprasad-c/k8s-dashboard/internal/ui/screens/clusterselect"
	"github.com/csivaprasad-c/k8s-dashboard/internal/ui/screens/connecting"
	"github.com/csivaprasad-c/k8s-dashboard/internal/ui/screens/dashboard"
)

type state int

const (
	stateClusterSelect state = iota
	stateConnecting
	stateDashboard
)

// App is the root Bubble Tea model. It owns which screen is active and
// routes messages to it, watching for the handful of transition messages
// screens emit to move between states.
type App struct {
	state state
	cs    clusterselect.Model
	conn  connecting.Model
	dash  dashboard.Model

	width, height int
}

func New() App {
	return App{cs: clusterselect.New()}
}

func (a App) Init() tea.Cmd {
	return a.cs.Init()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.cs.SetSize(msg.Width, msg.Height)
		a.dash.SetSize(msg.Width, msg.Height)
		return a, nil

	case tea.KeyMsg:
		// Global safety net: Ctrl+C always exits, no matter what's focused.
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}

	case clusterselect.QuitMsg:
		return a, tea.Quit

	case clusterselect.ContextChosenMsg:
		a.state = stateConnecting
		a.conn = connecting.New(msg.Context)
		return a, a.conn.Init()

	case connecting.ConnectedMsg:
		a.state = stateDashboard
		a.dash = dashboard.New(msg.Session, msg.Version)
		a.dash.SetSize(a.width, a.height)
		return a, a.dash.Init()

	case connecting.BackMsg:
		a.state = stateClusterSelect
		a.cs = clusterselect.New()
		a.cs.SetSize(a.width, a.height)
		return a, nil

	case dashboard.BackToClusterSelectMsg:
		a.state = stateClusterSelect
		a.cs = clusterselect.New()
		a.cs.SetSize(a.width, a.height)
		return a, nil
	}

	var cmd tea.Cmd
	switch a.state {
	case stateClusterSelect:
		a.cs, cmd = a.cs.Update(msg)
	case stateConnecting:
		a.conn, cmd = a.conn.Update(msg)
	case stateDashboard:
		a.dash, cmd = a.dash.Update(msg)
	}
	return a, cmd
}

func (a App) View() string {
	switch a.state {
	case stateClusterSelect:
		return a.cs.View()
	case stateConnecting:
		return a.conn.View()
	case stateDashboard:
		return a.dash.View()
	default:
		return ""
	}
}
