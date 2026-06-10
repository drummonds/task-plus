package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// SaveRemotes rewrites the remotes key in task-plus.yml, preserving all other
// keys and their comments. If releaseRemote is non-empty the release_remote
// key is set too; otherwise any existing release_remote is left untouched.
// Creates the file if it doesn't exist.
func SaveRemotes(dir string, remotes []Remote, releaseRemote string) error {
	if len(remotes) == 0 {
		return fmt.Errorf("cannot save empty remotes list")
	}

	path := filepath.Join(dir, configFile)
	var doc yaml.Node
	if data, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parsing %s: %w", configFile, err)
		}
	}
	if doc.Kind == 0 || len(doc.Content) == 0 {
		doc = yaml.Node{
			Kind:    yaml.DocumentNode,
			Content: []*yaml.Node{{Kind: yaml.MappingNode}},
		}
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return fmt.Errorf("%s: top level is not a mapping", configFile)
	}

	seq := &yaml.Node{Kind: yaml.SequenceNode}
	for _, r := range remotes {
		var n yaml.Node
		if err := n.Encode(r); err != nil {
			return err
		}
		seq.Content = append(seq.Content, &n)
	}
	setMapKey(root, "remotes", seq)

	if releaseRemote != "" {
		setMapKey(root, "release_remote", &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: releaseRemote,
		})
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0644)
}

// setMapKey replaces the value of key in a mapping node, or appends the pair.
func setMapKey(m *yaml.Node, key string, value *yaml.Node) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content[i+1] = value
			return
		}
	}
	m.Content = append(m.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		value,
	)
}
