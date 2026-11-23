package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	r := Recipe{Command: "git switch {{branch}} && echo {{port}}"}
	got, err := r.Render("sh", map[string]string{"branch": "feature/it's ready", "port": "8080"})
	if err != nil || got != "git switch 'feature/it'\"'\"'s ready' && echo 8080" {
		t.Fatalf("Render = %q, %v", got, err)
	}
	_, err = r.Render("sh", map[string]string{"branch": "main"})
	if err == nil {
		t.Fatal("missing port should prevent running")
	}
}

func TestBootstrapAndSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".cmdpal", "commands.yaml")
	store := Store{Path: path}
	recipes, err := store.Load()
	if err != nil || len(recipes) != 0 {
		t.Fatalf("initial Load = %v, %v", recipes, err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(raw), "#") {
		t.Fatalf("expected commented examples: %s, %v", raw, err)
	}
	if err := store.Add(Recipe{Name: "Go tests", Command: "go test ./..."}); err != nil {
		t.Fatal(err)
	}
	recipes, err = store.Load()
	if err != nil || len(recipes) != 1 || recipes[0].Name != "Go tests" || recipes[0].Safety != Caution {
		t.Fatalf("saved Load = %v, %v", recipes, err)
	}
}

func TestFuzzyFilter(t *testing.T) {
	recipes := []Recipe{{Name: "Tail container logs"}, {Name: "Git status"}, {Name: "Docker stats"}}
	found := Filter(recipes, "tcl")
	if len(found) != 1 || found[0].Name != "Tail container logs" {
		t.Fatalf("Filter = %v", found)
	}
}

func TestContextAndHistoryPins(t *testing.T) {
	catalog := Catalog()
	indices := Pinned(catalog, []string{"git:main", "docker", "go"}, []string{"git status", "git status"})
	if len(indices) < 5 || catalog[indices[0]].Name != "Working tree status" {
		t.Fatalf("history did not move Git status to the top: %v", indices)
	}
	categories := map[string]bool{}
	for _, index := range indices {
		categories[catalog[index].Category] = true
	}
	if !categories[Docker] || !categories[System] {
		t.Fatalf("a project marker was crowded out: %v", categories)
	}
}
