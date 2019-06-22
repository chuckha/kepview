package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/chuckha/kepctl/keps"
	"github.com/pkg/errors"
)

/*
	kepctl list
	<prints out list in most recent commit order>

	// V2
	kepctl search term term term
	# kep name
	...something <highlighted>term term</hilighted> something...

	...

	keptcl list -filter <filed>=<value> -filter ... -sort <field>
	<print list filtered by filters and sorted by field
*/

type filters []filter

func (f filters) String() string {
	out := ""
	for _, filter := range f {
		out += fmt.Sprintf("%s=%s", filter.field, filter.value)
	}
	return out
}
func (f *filters) Set(s string) error {
	if strings.Index(s, "=") == -1 {
		return errors.New("filter format is field=value, must contain an =")
	}
	fs := strings.Split(s, ",")
	for _, filter := range fs {
		split := strings.Split(strings.TrimSpace(filter), "=")
		*f = append(*f, newFilter(split[0], split[1]))
	}
	return nil
}

type filter struct {
	field, value string
}

func newFilter(field, value string) filter {
	return filter{field, value}
}

type config struct {
	root      string
	debug     bool
	filters   filters
	sortField string
}

func main() {
	configuration := &config{}
	list := flag.NewFlagSet("list", flag.ExitOnError)
	list.StringVar(&configuration.root, "keps", filepath.Join("enhancements", "keps"), "the location of the keps directory")
	list.BoolVar(&configuration.debug, "debug", false, "see debug logs")
	list.Var(&configuration.filters, "filters", "filter keps based on a fields and values (field=value,field2=value2)")
	list.StringVar(&configuration.sortField, "sort-by", "", "field to sort proposals by, descending")
	list.Parse(os.Args[1:])

	out := &keps.Proposals{}
	if err := filepath.Walk(configuration.root,
		FindEnhancements(out, &Opener{}, keps.NewParser(), &Logger{configuration.debug}, configuration.filters...)); err != nil {
		fmt.Printf("%+v", err)
		os.Exit(2)
	}
	out.SortBy(configuration.sortField)
	for _, proposal := range *out {
		fmt.Printf("%v: %v\n", proposal.CreationDate, proposal.Title)
	}
}

type Logger struct {
	debug bool
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	if l.debug {
		fmt.Printf(format, args...)
	}
}

type parser interface {
	Parse(io.Reader) (*keps.Proposal, error)
}

type opener interface {
	Open(string) (*os.File, error)
}

type Opener struct{}

func (o *Opener) Open(path string) (*os.File, error) {
	return os.Open(path)
}

type logger interface {
	Debugf(format string, args ...interface{})
}

// FindEnhancements will populate the out struct with any proposals found while
// walking the filesystem.
func FindEnhancements(out *keps.Proposals, opener opener, parser parser, log logger, filters ...filter) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrapf(err, "filename: %v", path)
		}
		// Ignore all README* files
		if strings.HasPrefix(info.Name(), "README") {
			return nil
		}
		// Ignore all non-markdown files
		if !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}
		// Ignore template files
		if strings.HasSuffix(info.Name(), "template.md") {
			return nil
		}
		file, err := opener.Open(path)
		if err != nil {
			return errors.Wrapf(err, "filename: %v", path)
		}
		defer file.Close()
		kep, err := parser.Parse(file)
		if err != nil {
			log.Debugf("Error parsing %q\n%v\n", path, err)
			// parsing errors are ok to skip.
			return nil
		}
		for _, filter := range filters {
			if !kep.Filter(filter.field, filter.value) {
				return nil
			}
		}
		kep.Filename = info.Name()
		out.AddProposal(kep)
		return nil
	}
}
