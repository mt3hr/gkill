package skills

import (
	"bytes"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Manifest は SKILL.md の frontmatter のうち gkill が使う項目。
// ほかの項目（license など Agent Skills の任意項目）は読み飛ばす。
type Manifest struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// ParseManifest は SKILL.md の先頭の frontmatter（"---" で挟んだ YAML）を読む。
// name と description の両方が必要で、expectedName が空でなければ name と一致しなければならない。
func ParseManifest(content []byte, expectedName string) (*Manifest, error) {
	text := string(bytes.TrimPrefix(content, utf8BOM))
	lines := strings.SplitAfter(text, "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], "\r\n") != "---" {
		return nil, detailError(ErrInvalidFrontmatter, `SKILL.md must start with a "---" line followed by YAML (name, description) and another "---" line`)
	}
	closing := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], "\r\n") == "---" {
			closing = i
			break
		}
	}
	if closing < 0 {
		return nil, detailError(ErrInvalidFrontmatter, `the closing "---" line of the frontmatter is missing`)
	}

	manifest := &Manifest{}
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:closing], "")), manifest); err != nil {
		return nil, detailError(ErrInvalidFrontmatter, fmt.Sprintf("YAML could not be parsed: %v", err))
	}
	manifest.Name = strings.TrimSpace(manifest.Name)
	manifest.Description = strings.TrimSpace(manifest.Description)
	if manifest.Name == "" {
		return nil, detailError(ErrInvalidFrontmatter, "name is empty")
	}
	if err := ValidateSkillName(manifest.Name); err != nil {
		return nil, detailError(ErrInvalidFrontmatter, DescribeError(err))
	}
	if expectedName != "" && manifest.Name != expectedName {
		return nil, detailError(ErrInvalidFrontmatter, fmt.Sprintf("name %q does not match the skill directory %q", manifest.Name, expectedName))
	}
	if manifest.Description == "" {
		return nil, detailError(ErrInvalidFrontmatter, "description is empty")
	}
	return manifest, nil
}

// BuildManifest は name / description / 本文から SKILL.md の中身を組み立てる。
// description は YAML の文字列として安全に書くため、yaml ライブラリに直列化させる。
func BuildManifest(name string, description string, body string) ([]byte, error) {
	header, err := yaml.Marshal(&Manifest{Name: name, Description: description})
	if err != nil {
		return nil, fmt.Errorf("error at marshal skill manifest: %w", err)
	}
	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(header)
	buf.WriteString("---\n")
	buf.WriteString(body)
	if body != "" && !strings.HasSuffix(body, "\n") {
		buf.WriteString("\n")
	}
	return buf.Bytes(), nil
}
