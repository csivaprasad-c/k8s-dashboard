package dashboard

import "github.com/csivaprasad/k8s-dashboard/internal/ui/styles"

func renderHelp() string {
	text := `:                jump to a resource tab (type to search)
tab / shift+tab  cycle resource tabs
↑/↓              move selection
/                filter current table
n                switch namespace
enter            view YAML
l                tail logs (pods only)
x                delete selected (with confirmation)
r                force refresh
ctrl+k           back to cluster picker
?                toggle this help
q / ctrl+c       quit`
	return styles.Border.Padding(1, 2).Render(styles.Title.Render("Keybindings") + "\n\n" + text + "\n\n" + styles.StatusBar.Render("press any key to close"))
}
