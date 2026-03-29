package team

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type TeamFile struct {
	Version int               `yaml:"version"`
	Members map[string]Member `yaml:"members"`
}

type Member struct {
	GitHub         string   `yaml:"github"`
	SSHFingerprint string   `yaml:"ssh_fingerprint"`
	AgePublicKey   string   `yaml:"age_public_key,omitempty"`
	SSHPublicKey   string   `yaml:"ssh_public_key,omitempty"`
	Added          string   `yaml:"added"`
	AddedBy        string   `yaml:"added_by"`
	Environments   []string `yaml:"environments,omitempty"`
}

func ReadTeamFile(path string) (*TeamFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var tf TeamFile
	if err := yaml.Unmarshal(data, &tf); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	if tf.Members == nil {
		tf.Members = make(map[string]Member)
	}
	return &tf, nil
}

func WriteTeamFile(path string, tf *TeamFile) error {
	data, err := yaml.Marshal(tf)
	if err != nil {
		return fmt.Errorf("marshal team file: %w", err)
	}

	header := "# envsync team manifest\n"
	content := append([]byte(header), data...)
	return os.WriteFile(path, content, 0644)
}

func NewTeamFile(username, sshFingerprint, sshPublicKey, addedBy string) *TeamFile {
	return &TeamFile{
		Version: 1,
		Members: map[string]Member{
			username: {
				GitHub:         username,
				SSHFingerprint: sshFingerprint,
				SSHPublicKey:   sshPublicKey,
				Added:          time.Now().UTC().Format("2006-01-02"),
				AddedBy:        addedBy,
			},
		},
	}
}

func (tf *TeamFile) AddMember(username, sshFingerprint, sshPublicKey, addedBy string) error {
	if _, exists := tf.Members[username]; exists {
		return fmt.Errorf("member %q already exists", username)
	}

	tf.Members[username] = Member{
		GitHub:         username,
		SSHFingerprint: sshFingerprint,
		SSHPublicKey:   sshPublicKey,
		Added:          time.Now().UTC().Format("2006-01-02"),
		AddedBy:        addedBy,
	}
	return nil
}

func (tf *TeamFile) RemoveMember(username string) error {
	if _, exists := tf.Members[username]; !exists {
		return fmt.Errorf("member %q not found", username)
	}
	delete(tf.Members, username)
	return nil
}

func (tf *TeamFile) GetPublicKeys() []string {
	var keys []string
	for _, m := range tf.Members {
		if m.SSHPublicKey != "" {
			keys = append(keys, m.SSHPublicKey)
		} else if m.AgePublicKey != "" {
			keys = append(keys, m.AgePublicKey)
		}
	}
	return keys
}

func (tf *TeamFile) UsesSSHKeys() bool {
	for _, m := range tf.Members {
		if m.SSHPublicKey != "" {
			return true
		}
	}
	return false
}

func (tf *TeamFile) GetSSHPublicKeys() []string {
	var keys []string
	for _, m := range tf.Members {
		if m.SSHPublicKey != "" {
			keys = append(keys, m.SSHPublicKey)
		}
	}
	return keys
}

func (tf *TeamFile) GetSSHPublicKeysForEnv(env string) []string {
	var keys []string
	for _, m := range tf.Members {
		if m.SSHPublicKey != "" && memberHasEnvAccess(m, env) {
			keys = append(keys, m.SSHPublicKey)
		}
	}
	return keys
}

func (tf *TeamFile) MemberNamesForEnv(env string) []string {
	var names []string
	for name, m := range tf.Members {
		if memberHasEnvAccess(m, env) {
			names = append(names, name)
		}
	}
	return names
}

func (tf *TeamFile) AddEnvToMember(username, env string) error {
	m, exists := tf.Members[username]
	if !exists {
		return fmt.Errorf("member %q not found", username)
	}
	for _, e := range m.Environments {
		if e == env {
			return nil
		}
	}
	m.Environments = append(m.Environments, env)
	tf.Members[username] = m
	return nil
}

func (tf *TeamFile) RemoveEnvFromMember(username, env string) error {
	m, exists := tf.Members[username]
	if !exists {
		return fmt.Errorf("member %q not found", username)
	}
	var filtered []string
	for _, e := range m.Environments {
		if e != env {
			filtered = append(filtered, e)
		}
	}
	m.Environments = filtered
	tf.Members[username] = m
	return nil
}

func memberHasEnvAccess(m Member, env string) bool {
	if env == "" || env == "development" {
		if len(m.Environments) == 0 {
			return true
		}
		for _, e := range m.Environments {
			if e == "" || e == "development" {
				return true
			}
		}
		return false
	}
	for _, e := range m.Environments {
		if e == env {
			return true
		}
	}
	return false
}

func (tf *TeamFile) GetMember(username string) (Member, bool) {
	m, ok := tf.Members[username]
	return m, ok
}
func (tf *TeamFile) MemberNames() []string {
	var names []string
	for name := range tf.Members {
		names = append(names, name)
	}
	return names
}
