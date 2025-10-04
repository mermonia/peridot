package state

import (
	"os"
	"testing"
	"time"

	"github.com/mermonia/peridot/internal/tree"
)

func TestGetStateFileTree(t *testing.T) {
	state := &State{
		Modules: map[string]*ModuleState{
			"hyprland": {
				DeployedAt: time.Now(),
				Files: map[string]*Entry{
					".config/hypr/hyprland.conf": {},
				},
			},
			"waybar": {
				DeployedAt: time.Now(),
				Files: map[string]*Entry{
					".config/waybar/config.jsonc":       {},
					".config/waybar/style.css":          {},
					".config/waybar/scripts/spotify.sh": {},
				},
			},
		},
	}

	tr, err := GetStateFileTree(state)
	if err != nil {
		t.Fatalf("GetStateFileTree should not return an error")
	}

	tree.PrintTree(tr, tree.DefaultTreeBranchSymbols, os.Stdout)
}
