package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const initial = `# Add commands here or press n inside cmdpal.
# Use {{name}} for an argument you want to fill in before running.
#
# commands:
#   - name: Show Git history
#     command: git log --oneline -n {{count}}
#     summary: Show one line per commit.
#     safety: SAFE
#     defaults:
#       count: "10"
#     flags:
#       - name: --oneline
#         description: Show the short hash and subject.
#       - name: -n
#         description: Limit the number of commits.
commands: []
`

type Store struct {
	Path string
}

func (s Store) Load() ([]Recipe, error) {
	data, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(s.Path), 0700); err != nil {
			return nil, err
		}
		file, createErr := os.OpenFile(s.Path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if createErr == nil {
			_, writeErr := io.WriteString(file, initial)
			createErr = errors.Join(writeErr, file.Close())
		}
		if createErr != nil && !errors.Is(createErr, os.ErrExist) {
			return nil, createErr
		}
		data, err = os.ReadFile(s.Path)
	}
	if err != nil {
		return nil, err
	}
	var document struct{ Commands []Recipe }
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&document); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("read %s: %w", s.Path, err)
	}
	for index, recipe := range document.Commands {
		if err := normalize(&recipe); err != nil {
			return nil, fmt.Errorf("command %d: %w", index+1, err)
		}
		document.Commands[index] = recipe
	}
	return document.Commands, nil
}

func normalize(r *Recipe) error {
	r.Name = strings.TrimSpace(r.Name)
	r.Command = strings.TrimSpace(r.Command)
	if r.Name == "" || r.Command == "" {
		return errors.New("enter a name and command")
	}
	if strings.ContainsAny(r.Command+r.Windows, "\r\n\x00\x1b") {
		return errors.New("commands must fit on one line")
	}
	r.Category = "CUSTOM"
	r.Safety = strings.ToUpper(strings.TrimSpace(r.Safety))
	switch r.Safety {
	case "":
		r.Safety = Caution
	case Safe, Caution, Destructive:
	default:
		return fmt.Errorf("unknown safety level %q", r.Safety)
	}
	return nil
}

func (s Store) Add(recipe Recipe) error {
	if err := normalize(&recipe); err != nil {
		return err
	}
	recipes, err := s.Load()
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(s.Path)
	if err != nil {
		return err
	}
	var document yaml.Node
	if err := yaml.Unmarshal(raw, &document); err != nil {
		return err
	}
	if len(document.Content) == 0 {
		if err := yaml.Unmarshal([]byte(initial), &document); err != nil {
			return err
		}
	}
	root := document.Content[0]
	var sequence *yaml.Node
	for i := 0; i < len(root.Content); i += 2 {
		if root.Content[i].Value == "commands" {
			sequence = root.Content[i+1]
			break
		}
	}
	if sequence == nil {
		root.Kind, root.Tag, root.Value = yaml.MappingNode, "!!map", ""
		sequence = &yaml.Node{}
		root.Content = append(root.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "commands"}, sequence)
	}
	sequence.Kind, sequence.Tag, sequence.Style, sequence.Value = yaml.SequenceNode, "!!seq", 0, ""
	if len(sequence.Content) != len(recipes) {
		return errors.New("commands must be a YAML list")
	}
	var entry yaml.Node
	if err := entry.Encode(recipe); err != nil {
		return err
	}
	sequence.Content = append(sequence.Content, &entry)
	temp, err := os.CreateTemp(filepath.Dir(s.Path), ".commands-*.yaml")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	encoder := yaml.NewEncoder(temp)
	encoder.SetIndent(2)
	if err := errors.Join(encoder.Encode(&document), encoder.Close(), temp.Close()); err != nil {
		return err
	}
	return os.Rename(temp.Name(), s.Path)
}
