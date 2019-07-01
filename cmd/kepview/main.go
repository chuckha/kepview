/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/chuckha/kepview/keps"
	"github.com/pkg/errors"
)

type config struct {
	root      string
	debug     bool
	sortField string
	validate  bool
}

func main() {
	configuration := &config{}
	list := flag.NewFlagSet("list", flag.ExitOnError)
	list.StringVar(&configuration.root, "keps", ".", "the location of the keps directory")
	list.BoolVar(&configuration.debug, "debug", false, "see debug logs")
	list.BoolVar(&configuration.validate, "validate-only", false, "only run the metadata validations")
	list.Parse(os.Args[1:])

	ef := NewEnhancementFinder(
		WithLog(&Logger{configuration.debug}),
	)

	out := &keps.Proposals{}
	if err := filepath.Walk(configuration.root, ef.Find(out)); err != nil {
		fmt.Printf("%+v", err)
		os.Exit(2)
	}
	exit := 0
	for _, proposal := range *out {
		if proposal.Error != nil {
			fmt.Printf("%s has an error: %q\n", proposal.Filename, proposal.Error)
			exit = 1
		}

		if configuration.validate {
			continue
		}
		// TODO: do something interesting here, perhaps aggregation or sorting.
		fmt.Printf("%v\n", proposal.Filename)
	}
	if exit == 0 && configuration.validate {
		fmt.Println("No validation errors")
	}
	os.Exit(exit)
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
	Parse(io.Reader) *keps.Proposal
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

func defaultFilters() []filter {
	return []filter{
		filenameFilter{
			func(in string) bool {
				return strings.HasPrefix(in, "README")
			},
			"Ignore READMEs",
		},
		filenameFilter{
			func(in string) bool {
				return !strings.HasSuffix(in, ".md")
			},
			"Ignore non markdown files",
		},
		filenameFilter{
			func(in string) bool {
				return strings.HasSuffix(in, "template.md")
			},
			"Ignore template files",
		},
		filenameFilter{
			func(in string) bool {
				return in == "kep-faq.md"
			},
			"Ignore the kep faq",
		},
	}
}

type filter interface {
	Filter(string) bool
}
type filenameFilter struct {
	f    func(string) bool
	name string
}

func (f filenameFilter) Filter(in string) bool {
	return f.f(in)
}
func (f filenameFilter) String() string {
	return f.name
}

// EnhancementFinder can filter out non-enhancement-like filenames in
// addition to parsing the KEPs and reporting failure statuses
type EnhancementFinder struct {
	opener          opener
	parser          parser
	filenameFilters []filter
	log             logger
}

// NewEnhancementFinder returns a reasonably configured EnhancementFinder
func NewEnhancementFinder(opts ...finderOpts) *EnhancementFinder {
	ef := &EnhancementFinder{
		opener:          &Opener{},
		parser:          &keps.Parser{},
		log:             &Logger{},
		filenameFilters: defaultFilters(),
	}
	for _, opt := range opts {
		opt(ef)
	}
	return ef
}

type finderOpts func(*EnhancementFinder)

// WithOpener sets the object that opens files
func WithOpener(opener opener) finderOpts {
	return func(e *EnhancementFinder) { e.opener = opener }
}

// WithParser sets the parser that prases KEPs
func WithParser(parser parser) finderOpts {
	return func(e *EnhancementFinder) { e.parser = parser }
}

// WithLog defines the logger for the finder
func WithLog(log logger) finderOpts {
	return func(e *EnhancementFinder) { e.log = log }
}

// WithFilenameFilters sets the list of filters the filenames must pass
func WithFilenameFilters(filters ...filter) finderOpts {
	return func(e *EnhancementFinder) { e.filenameFilters = filters }
}

// Find returns a function that filters out filenames and prases a valid KEP file.
// Is also a WalkFunc.
func (e *EnhancementFinder) Find(out *keps.Proposals) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrapf(err, "filename: %v", path)
		}
		for _, f := range e.filenameFilters {
			if f.Filter(info.Name()) {
				e.log.Debugf("Skipping %q due to filename filter: %v\n", info.Name(), f)
				return nil
			}
		}
		file, err := e.opener.Open(path)
		if err != nil {
			return errors.Wrapf(err, "filename: %v", path)
		}
		defer file.Close()
		// Parse always returns a proposal even on failure.
		kep := e.parser.Parse(file)
		kep.Filename = path
		out.AddProposal(kep)
		return nil
	}
}
