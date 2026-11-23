package context

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type Project struct {
	Dir     string
	Badges  []string
	Branch  string
	entries []os.DirEntry
	gitDir  string
}

func Detect(dir string) (Project, error) {
	p := Project{Dir: dir}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return p, err
	}
	p.entries = entries
	markers := make(map[string]bool, len(entries))
	for _, e := range entries {
		markers[strings.ToLower(e.Name())] = true
	}
	for parent := dir; ; parent = filepath.Dir(parent) {
		if parent != dir || markers[".git"] {
			p.gitDir = resolveGitDir(parent)
		}
		if p.gitDir != "" || parent == filepath.Dir(parent) {
			break
		}
	}
	if p.gitDir != "" {
		head, _ := os.ReadFile(filepath.Join(p.gitDir, "HEAD"))
		raw := strings.TrimSpace(string(head))
		if branch, ok := strings.CutPrefix(raw, "ref: refs/heads/"); ok {
			p.Branch = branch
			p.Badges = append(p.Badges, "git:"+branch)
		} else if len(raw) >= 7 {
			p.Badges = append(p.Badges, "git:"+raw[:7])
		} else {
			p.Badges = append(p.Badges, "git")
		}
	}
	for _, group := range [][]string{
		{"node", "package.json"}, {"go", "go.mod"}, {"rust", "cargo.toml"},
		{"docker", "dockerfile", "compose.yml", "compose.yaml", "docker-compose.yml", "docker-compose.yaml"},
		{"python", "pyproject.toml", "requirements.txt"}, {"make", "makefile"},
	} {
		if slices.ContainsFunc(group[1:], func(name string) bool { return markers[name] }) {
			p.Badges = append(p.Badges, group[0])
		}
	}
	return p, nil
}

func resolveGitDir(dir string) string {
	path := filepath.Join(dir, ".git")
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}
	if info.IsDir() {
		return path
	}
	raw, _ := os.ReadFile(path)
	gitdir, ok := strings.CutPrefix(strings.TrimSpace(string(raw)), "gitdir: ")
	if !ok {
		return ""
	}
	if filepath.IsAbs(gitdir) {
		return gitdir
	}
	return filepath.Join(dir, gitdir)
}

func (p Project) Branches() []string {
	if p.gitDir == "" {
		return nil
	}
	common := p.gitDir
	if raw, err := os.ReadFile(filepath.Join(common, "commondir")); err == nil {
		common = strings.TrimSpace(string(raw))
		if !filepath.IsAbs(common) {
			common = filepath.Join(p.gitDir, common)
		}
	}
	root := filepath.Join(common, "refs", "heads")
	var branches []string
	_ = filepath.WalkDir(root, func(path string, info fs.DirEntry, err error) error {
		if err == nil && !info.IsDir() && !strings.HasSuffix(path, ".lock") {
			name, _ := filepath.Rel(root, path)
			branches = append(branches, filepath.ToSlash(name))
		}
		return nil
	})
	if file, err := os.Open(filepath.Join(common, "packed-refs")); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			_, ref, _ := strings.Cut(scanner.Text(), " ")
			if branch, ok := strings.CutPrefix(ref, "refs/heads/"); ok {
				branches = append(branches, branch)
			}
		}
	}
	slices.Sort(branches)
	return slices.Compact(branches)
}

func (p Project) Files() []string {
	var files []string
	for _, entry := range p.entries {
		if !entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			files = append(files, entry.Name())
		}
	}
	return files
}
