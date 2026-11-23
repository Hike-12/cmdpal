package config

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

const (
	Safe        = "SAFE"
	Caution     = "CAUTION"
	Destructive = "DESTRUCTIVE"
)

type Flag struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type Recipe struct {
	Name     string            `yaml:"name"`
	Command  string            `yaml:"command"`
	Windows  string            `yaml:"windows,omitempty"`
	Summary  string            `yaml:"summary,omitempty"`
	Safety   string            `yaml:"safety,omitempty"`
	Flags    []Flag            `yaml:"flags,omitempty"`
	Defaults map[string]string `yaml:"defaults,omitempty"`
	Category string            `yaml:"-"`
	Tags     []string          `yaml:"-"`
}

var slot = regexp.MustCompile(`\{\{([a-zA-Z][a-zA-Z0-9_-]*)\}\}`)
var bare = regexp.MustCompile(`^[a-zA-Z0-9_./:@+,=-]+$`)
var cmdArg = regexp.MustCompile(`^[a-zA-Z0-9_./:@+,= \\-]+$`)

func (r Recipe) Template(shell string) string {
	if (shell == "pwsh" || shell == "cmd") && r.Windows != "" {
		return r.Windows
	}
	return r.Command
}

func (r Recipe) Params(shell string) []string {
	var names []string
	for _, match := range slot.FindAllStringSubmatch(r.Template(shell), -1) {
		if !slices.Contains(names, match[1]) {
			names = append(names, match[1])
		}
	}
	return names
}

func (r Recipe) Preview(shell string, values map[string]string) string {
	command, _ := r.render(shell, values)
	return command
}

func (r Recipe) Render(shell string, values map[string]string) (string, error) {
	return r.render(shell, values)
}

func (r Recipe) render(shell string, values map[string]string) (string, error) {
	var first error
	command := slot.ReplaceAllStringFunc(r.Template(shell), func(token string) string {
		name := token[2 : len(token)-2]
		value := values[name]
		var err error
		switch name {
		case "port", "pid", "count", "crf":
			number, parseErr := strconv.ParseUint(value, 10, 31)
			if parseErr != nil {
				err = fmt.Errorf("%s must be a whole number", name)
			} else if name == "port" && (number == 0 || number > 65535) {
				err = fmt.Errorf("port must be between 1 and 65535")
			} else if (name == "pid" || name == "count") && number == 0 {
				err = fmt.Errorf("%s must be greater than zero", name)
			} else if name == "crf" && number > 51 {
				err = fmt.Errorf("crf must be between 0 and 51")
			}
		default:
			if value == "" {
				err = fmt.Errorf("enter %s", name)
			}
		}
		var word string
		if err == nil {
			word, err = quote(shell, value)
		}
		if err != nil {
			if first == nil {
				first = err
			}
			return token
		}
		return word
	})
	return command, first
}

func quote(shell, value string) (string, error) {
	if strings.ContainsFunc(value, unicode.IsControl) {
		return "", fmt.Errorf("arguments cannot contain control characters")
	}
	if shell == "cmd" {
		if !cmdArg.MatchString(value) {
			return "", fmt.Errorf("this argument needs PowerShell")
		}
		if strings.ContainsRune(value, ' ') {
			return `"` + value + `"`, nil
		}
		return value, nil
	}
	if bare.MatchString(value) {
		return value, nil
	}
	if shell == "pwsh" {
		return "'" + strings.NewReplacer("'", "''", "‘", "‘‘", "’", "’’").Replace(value) + "'", nil
	}
	if shell == "fish" {
		return "'" + strings.NewReplacer("\\", "\\\\", "'", "\\'").Replace(value) + "'", nil
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'", nil
}

func Filter(recipes []Recipe, query string) []Recipe {
	var found []Recipe
	for _, recipe := range recipes {
		if MatchScore(recipe, query) > 0 {
			found = append(found, recipe)
		}
	}
	slices.SortStableFunc(found, func(a, b Recipe) int { return MatchScore(b, query) - MatchScore(a, query) })
	return found
}

func MatchScore(recipe Recipe, query string) int {
	query = strings.ToLower(strings.TrimSpace(query))
	name := strings.ToLower(recipe.Name)
	fields := strings.ToLower(recipe.Category + " " + recipe.Command + " " + recipe.Windows)
	score := 1
	if query != "" && strings.Contains(name, query) {
		score += 10
	}
	for _, token := range strings.Fields(query) {
		switch {
		case strings.Contains(name, token):
			score += 3
		case strings.Contains(fields, token):
			score += 2
		case subsequence(name, token):
			score++
		default:
			return 0
		}
	}
	return score
}

func subsequence(text, token string) bool {
	want := []rune(token)
	for _, char := range text {
		if char == want[0] {
			want = want[1:]
			if len(want) == 0 {
				return true
			}
		}
	}
	return false
}
