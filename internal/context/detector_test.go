package context

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestDetect(t *testing.T) {
	dir := t.TempDir()
	write := func(path, text string) {
		t.Helper()
		path = filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(".git/HEAD", "ref: refs/heads/main\n")
	write(".git/refs/heads/topic", "9a27b")
	write("package.json", "{}")
	write("go.mod", "module example")
	write("Dockerfile", "FROM scratch")
	project, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"git:main", "node", "go", "docker"}
	if !reflect.DeepEqual(project.Badges, want) || !slices.Contains(project.Branches(), "topic") {
		t.Fatalf("Detect = %v, branches %v", project.Badges, project.Branches())
	}
}

func TestParseHistory(t *testing.T) {
	tests := []struct{ shell, input string }{
		{"zsh", ": 1715555000:0;git status\n: 1715555001:0;go test ./...\n"},
		{"bash", "#1715555000\ngit status\ngo test ./...\n"},
		{"pwsh", "git status\r\ngo test ./...\r\n"},
	}
	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			got, err := ParseHistory(strings.NewReader(tt.input), tt.shell)
			if err != nil || !slices.Equal(got, []string{"git status", "go test ./..."}) {
				t.Fatalf("ParseHistory = %v, %v", got, err)
			}
		})
	}
}
