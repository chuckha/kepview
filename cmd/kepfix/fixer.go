package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/chuckha/kepview/keps"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func main() {
	var dryRun bool
	flag.BoolVar(&dryRun, "dry-run", false, "edit files in place")
	flag.Parse()
	for _, path := range flag.Args() {
		if err := FixYAML(path, dryRun); err != nil {
			fmt.Printf("%+v", err)
			os.Exit(1)
		}
	}
}

var errRe = regexp.MustCompile(`line (\d+): cannot unmarshal !!map into string`)
var unexpectedHyphenRe = regexp.MustCompile(`line (\d+): did not find expected '-' indicator`)
var atsignRe = regexp.MustCompile(`line (\d+): found character that cannot start any token`)
var keyFindRe = regexp.MustCompile(`\s*[a-z]+:`)

func fixMapInListContext(path string, dryRun bool) error {
	proposal, head, meta, body, err := openProposal(path)
	if err != nil {
		return err
	}
	if proposal == nil {
		return nil
	}
	if proposal.Error == nil {
		return nil
	}
	lines := bytes.Split(meta, []byte("\n"))

	// using an object in a list context
	matches := errRe.FindAllStringSubmatch(proposal.Error.Error(), -1)
	for _, match := range matches {
		lineNumber, err := strconv.Atoi(match[1])
		if err != nil {
			fmt.Printf("ERROR CONVERTING INT: %v\n", lineNumber)
		}
		replaced := keyFindRe.ReplaceAllLiteral(lines[lineNumber-1], []byte(""))
		if bytes.IndexByte(replaced, '-') < 0 {
			lines[lineNumber-2] = append(lines[lineNumber-2], replaced...)
			lines = append(lines[:lineNumber-1], lines[lineNumber:]...)
		} else {
			lines[lineNumber-1] = replaced
		}
	}
	if dryRun {
		return nil
	}
	if err := writeFile(path, head, bytes.Join(lines, []byte("\n")), body); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func fixBareAtSign(path string, dryRun bool) error {
	proposal, head, meta, body, err := openProposal(path)
	if err != nil {
		return err
	}
	if proposal == nil || proposal.Error == nil {
		return nil
	}
	lines := bytes.Split(meta, []byte("\n"))

	// using an object in a list context
	matches := atsignRe.FindAllStringSubmatch(proposal.Error.Error(), -1)
	for _, match := range matches {
		lineNumber, err := strconv.Atoi(match[1])
		if err != nil {
			fmt.Printf("ERROR CONVERTING INT (3): %v\n", lineNumber)
		}
		edited := bytes.Replace(lines[lineNumber-1], []byte("@"), []byte(`"@`), 1)
		edited = append(edited, []byte(`"`)...)
		lines[lineNumber-1] = edited
	}

	if dryRun {
		return nil
	}
	if err := writeFile(path, head, bytes.Join(lines, []byte("\n")), body); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func fixRawMarkdown(path string, dryRun bool) error {
	proposal, head, meta, body, err := openProposal(path)
	if err != nil {
		return err
	}
	if proposal == nil || proposal.Error == nil {
		return nil
	}
	lines := bytes.Split(meta, []byte("\n"))

	// markdown in raw yaml list...
	matches := unexpectedHyphenRe.FindAllStringSubmatch(proposal.Error.Error(), -1)
	for _, match := range matches {
		lineNumber, err := strconv.Atoi(match[1])
		if err != nil {
			fmt.Printf("ERROR CONVERTING INT (2): %v\n", lineNumber)
		}
		edited := bytes.Replace(lines[lineNumber], []byte("["), []byte(`"[`), 1)
		edited = append(edited, []byte(`"`)...)
		lines[lineNumber] = edited
	}

	if dryRun {
		return nil
	}
	if err := writeFile(path, head, bytes.Join(lines, []byte("\n")), body); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func FixYAML(path string, dryRun bool) error {
	if err := fixMapInListContext(path, dryRun); err != nil {
		return err
	}
	if err := fixBareAtSign(path, dryRun); err != nil {
		return err
	}
	if err := fixRawMarkdown(path, dryRun); err != nil {
		return err
	}

	if dryRun {
		return nil
	}

	// if we recognize the error then write out the good bytes followed by the
	// rest of the file
	// recursively call itself
	return nil
}

func writeFile(path string, head, meta, body []byte) error {

	w, err := os.OpenFile(path, os.O_RDWR|os.O_TRUNC, os.ModeAppend)
	if err != nil {
		return errors.WithStack(err)
	}
	defer w.Close()
	if len(head) > 0 {
		fmt.Fprint(w, string(head))
	}
	fmt.Fprintln(w, "---")
	fmt.Fprint(w, string(meta))
	fmt.Fprintln(w, "---")

	fmt.Fprint(w, string(body))
	return nil
}

func extractData(reader io.Reader) ([]byte, []byte, []byte, error) {
	scanner := bufio.NewScanner(reader)
	count := 0
	aboveTheHeader := []byte{}
	metadata := []byte{}
	restOfFile := []byte{}

	whereAmI := "start"

	// assume that the top of the file is the only yaml we want
	for scanner.Scan() {
		line := scanner.Text() + "\n"

		if (count < 2 && strings.HasPrefix(line, "---")) ||
			(count == 1 && strings.HasSuffix(strings.TrimSpace(line), "```")) {
			count++
			whereAmI = "metadata"
			if count == 2 {
				whereAmI = "body"
			}
			if strings.HasSuffix(strings.TrimSpace(line), "```") {
				restOfFile = []byte("```\n")
			}
			continue
		}

		if whereAmI == "start" {
			aboveTheHeader = append(aboveTheHeader, []byte(line)...)
		} else if whereAmI == "metadata" {
			metadata = append(metadata, []byte(line)...)
		} else {
			restOfFile = append(restOfFile, []byte(line)...)
		}
	}
	if count != 2 {
		return nil, nil, nil, errors.New("skip")
	}
	return aboveTheHeader, metadata, restOfFile, scanner.Err()
}

func openProposal(path string) (*keps.Proposal, []byte, []byte, []byte, error) {
	// open file for writing
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, nil, nil, errors.WithStack(err)
	}
	defer f.Close()
	// read yaml into bytes
	// read the rest of the file into bytes
	head, metadata, body, err := extractData(f)
	if err != nil {
		if err.Error() == "skip" {
			return nil, nil, nil, nil, nil
		}
		return nil, nil, nil, nil, errors.WithStack(err)
	}

	// parse yaml
	proposal := &keps.Proposal{}
	proposal.Error = yaml.Unmarshal(metadata, proposal)

	return proposal, head, metadata, body, err
}
