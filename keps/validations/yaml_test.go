package validations

import (
	"bytes"
	"html/template"
	"testing"

	"gopkg.in/yaml.v2"
)

type proposal struct {
	Title             string   `yaml:"title"`
	Authors           []string `yaml:,flow`
	OwningSIG         string   `yaml:"owning-sig"`
	ParticipatingSIGs []string `yaml:"participating-sigs",flow`
	Reviewers         []string `yaml:,flow`
	Approvers         []string `yaml:,flow`
	Editor            string   `yaml:"editor"`
	CreationDate      string   `yaml:"creation-date"`
	LastUpdated       string   `yaml:"last-updated"`
	Status            string   `yaml:"status"`
	SeeAlso           []string `yaml:"see-also"`
	Replaces          []string `yaml:"replaces"`
	SupersededBy      []string `yaml:"superseded-by"`
}

// YAMLDoc is entirely for testing purposes
func (p *proposal) YAMLDoc() []byte {
	t, err := template.New("yaml").Parse(`title: {{.Title}}
authors:
  {{- range .Authors }}
  - "{{.}}"
  {{- end }}
owning-sig: {{ .OwningSIG }}
{{- if .ParticipatingSIGs }}
participating-sigs:
  {{- range .ParticipatingSIGs }}
  - "{{.}}"
  {{- end }}
{{- end }}
reviewers:
  {{- range .Reviewers }}
  - "{{.}}"
  {{- end }}
approvers:
  {{- range .Approvers }}
  - "{{.}}"
  {{- end }}
{{- if .Editor }}
editor: {{ .Editor }}
{{- end }}
creation-date: {{ .CreationDate }}
last-updated: {{ .LastUpdated }}
status: {{ .Status }}
{{- if .SeeAlso }}
see-also:
  {{- range .SeeAlso }}
  - "{{.}}"
  {{- end }}
{{- end }}
{{- if .Replaces }}
replaces:
  {{- range .Replaces }}
  - "{{.}}"
  {{- end }}
{{- end }}
{{- if .SupersededBy }}
superseded-by:
  {{- range .SupersededBy }}
  - "{{.}}"
  {{- end }}
{{- end }}
`)
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, p); err != nil {
		panic(err)
	}
	return buf.Bytes()
}
func TestUnmarshal(t *testing.T) {
	yamlDoc := &proposal{
		Title:        "test",
		Authors:      []string{"test", "test", "test"},
		Reviewers:    []string{"my reviewer"},
		OwningSIG:    "my-sig",
		Status:       "some status",
		Approvers:    []string{"my approvers"},
		LastUpdated:  "at some point",
		CreationDate: "a while ago",
	}
	p := map[interface{}]interface{}{}

	if err := yaml.Unmarshal(yamlDoc.YAMLDoc(), p); err != nil {
		t.Fatal(err)
	}
	if err := ValidateStructure(p); err != nil {
		t.Fatal(err)
	}
}
