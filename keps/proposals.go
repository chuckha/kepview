package keps

import (
	"bufio"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/chuckha/kepview/keps/validations"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Proposals []*Proposal

type ByCreationDate Proposals

func (b ByCreationDate) Len() int           { return len(b) }
func (b ByCreationDate) Less(i, j int) bool { return b[i].CreationDate.After(b[j].CreationDate) }
func (b ByCreationDate) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

type ByTitle Proposals

func (b ByTitle) Len() int           { return len(b) }
func (b ByTitle) Less(i, j int) bool { return b[i].Title > b[j].Title }
func (b ByTitle) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

func (p *Proposals) AddProposal(proposal *Proposal) {
	*p = append(*p, proposal)
}
func (p *Proposals) SortBy(field string) {
	switch field {
	case "created", "creation", "creationDate":
		sort.Sort(ByCreationDate(*p))
	case "title":
		sort.Sort(ByTitle(*p))
	}
}

type Proposal struct {
	Title             string
	Authors           []string  `yaml:,flow`
	OwningSIG         string    `yaml:"owning-sig"`
	ParticipatingSIGs []string  `yaml:"participating-sigs",flow`
	Reviewers         []string  `yaml:,flow`
	Approvers         []string  `yaml:,flow`
	CreationDate      time.Time `yaml:"creation-date"`
	LastUpdated       time.Time `yaml:"last-updated"`
	Status            string
	SeeAlso           []string `yaml:"see-also"`

	Filename string `yaml:"-"`
}

func (p *Proposal) Filter(key, value string) bool {
	switch key {
	case "author":
		return Contains(p.Authors, value)
	case "status":
		return p.Status == value
	}
	return false
}

func Contains(haystack []string, needle string) bool {
	for _, hay := range haystack {
		if hay == needle {
			return true
		}
	}
	return false
}

type Parser struct {
}

func NewParser() *Parser {
	return &Parser{}
}

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
	if err := scanner.Err(); err != nil {
		return nil, errors.WithStack(err)
	}
	// First do structural checks
	test := map[interface{}]interface{}{}
	if err := yaml.Unmarshal(metadata, test); err != nil {
		return nil, errors.WithStack(err)
	}
	if err := validations.ValidateStructure(test); err != nil {
		return nil, errors.WithStack(err)
	}

	proposal := &Proposal{}
	err := yaml.Unmarshal(metadata, proposal)
	return proposal, errors.WithStack(err)
}
