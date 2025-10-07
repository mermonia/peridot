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
					".config/hypr/hyprland.conf": {
						Status: Synced,
						Target: "/home/mermonia/.config/hypr/hyprland",
					},
				},
				Status: Synced,
			},
			"waybar": {
				DeployedAt: time.Now(),
				Files: map[string]*Entry{
					".config/waybar/config.jsonc": {
						Status: Synced,
						Target: "/home/mermonia/.config/waybar/config.jsonc",
					},
					".config/waybar/style.css": {
						Status: Synced,
						Target: "/home/mermonia/.config/waybar/style.css",
					},
					".config/waybar/scripts/spotify.sh": {
						Status: Unsynced,
						Target: "/home/mermonia/.config/waybar/spotify.sh",
					},
				},
				Status: Unsynced,
			},
		},
	}

	tr, err := GetStateFileTree(state)
	if err != nil {
		t.Fatalf("GetStateFileTree should not return an error")
	}

	tree.PrintTree(tr, tree.DefaultTreeBranchSymbols, os.Stdout)

	tr, err = GetModuleFileTree("hyprland", &ModuleState{
		DeployedAt: time.Now(),
		Files: map[string]*Entry{
			".config/hypr/hyprland.conf": {},
		},
	})

	tree.PrintTree(tr, tree.DefaultTreeBranchSymbols, os.Stdout)
}
