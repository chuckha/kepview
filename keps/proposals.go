package keps

import (
	"bufio"
	"io"
	"strings"
	"time"

	"github.com/chuckha/kepview/keps/validations"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Proposals []*Proposal

func (p *Proposals) AddProposal(proposal *Proposal) {
	*p = append(*p, proposal)
}

type Proposal struct {
	Title             string    `yaml:"title"`
	Authors           []string  `yaml:,flow`
	OwningSIG         string    `yaml:"owning-sig"`
	ParticipatingSIGs []string  `yaml:"participating-sigs",flow`
	Reviewers         []string  `yaml:,flow`
	Approvers         []string  `yaml:,flow`
	Editor            string    `yaml:"editor"`
	CreationDate      time.Time `yaml:"creation-date"`
	LastUpdated       time.Time `yaml:"last-updated"`
	Status            string    `yaml:"status"`
	SeeAlso           []string  `yaml:"see-also"`
	Replaces          []string  `yaml:"replaces"`
	SupersededBy      []string  `yaml:"superseded-by"`

	Filename        string `yaml:"-"`
	ValidationError error  `yaml:"-"`
}

type Parser struct{}

func (p *Parser) Parse(in io.Reader) (*Proposal, error) {
	scanner := bufio.NewScanner(in)
	count := 0
	metadata := []byte{}
	for scanner.Scan() {
		line := scanner.Text() + "\n"
		if strings.Contains(line, "---") {
			count++
			continue
		}
		if count == 2 {
			break
		}
		metadata = append(metadata, []byte(line)...)

	}
	proposal := &Proposal{}
	if err := scanner.Err(); err != nil {
		return proposal, errors.WithStack(err)
	}

	// First do structural checks
	test := map[interface{}]interface{}{}
	if err := yaml.Unmarshal(metadata, test); err != nil {
		return proposal, errors.WithStack(err)
	}
	if err := validations.ValidateStructure(test); err != nil {
		return proposal, errors.WithStack(err)
	}

	err := yaml.Unmarshal(metadata, proposal)
	return proposal, errors.WithStack(err)
}
