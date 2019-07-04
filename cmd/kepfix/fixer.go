package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/chuckha/kepview/keps"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	kepRepo = "/Users/cha/go/src/k8s.io/enhancements"
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
		if err := FixData(path, dryRun); err != nil {
			fmt.Printf("%q\n%+v", path, err)
			os.Exit(1)
		}
	}
}

func requiredKeys() map[string]bool {
	return map[string]bool{
		"title":         false,
		"authors":       false,
		"reviewers":     false,
		"approvers":     false,
		"creation-date": false,
		"last-updated":  false,
		"status":        false,
	}
}

var timeRe = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)

type metadata []string

func (m *metadata) InsertLine(line string, at int) {
	out := make([]string, len(*m)+1)
	copy(out[:at], (*m)[:at])
	out[at] = line
	copy(out[at+1:], (*m)[at:])
	*m = out
}

var oneLineFix = regexp.MustCompile(`(\s*[[:alpha:]]*:\s*).*$`)

func (m *metadata) SetFieldValue(key string, val interface{}) error {
	start, end := m.Field(key)
	if start == end {
		match := oneLineFix.FindString((*m)[start])
		if match == "" {
			return errors.New("failed to match oneLineFix regex")
		}
		(*m)[start] = match
		item, ok := val.(string)
		if ok {
			(*m)[start] += item
			return nil
		}
		itemList := val.([]string)
		if ok {
			for range itemList {

			}
		}
	}
	return nil
}

// returns the linenumber start and end, if they are the same it's a one liner
func (m *metadata) Field(key string) (int, int) {
	start := -1
	end := -1
	for i, line := range *m {
		if strings.HasPrefix(strings.TrimSpace(line), key) {
			start, end = i, i
			continue
		}
		if start >= 0 && strings.Contains(strings.TrimSpace(line), ":") {
			end = i - 1
			return start, end
		}
	}
	return start, end
}

func getCreatedTime(path string) (string, error) {
	cmd := exec.Command("git", "log", "--diff-filter=A", "--follow", "--format=%ad", "-1", "--", path)
	cmd.Dir = kepRepo
	t, err := cmd.Output()
	if err != nil {
		return "", errors.WithStack(err)
	}
	t = bytes.Trim(bytes.TrimSpace(t), `"`)
	lastUpdate, err := time.Parse("Mon Jan 2 15:04:05 2006 -0700", string(t))
	if err != nil {
		return "", errors.WithStack(err)
	}
	return lastUpdate.Format("2006-01-02"), nil
}

func getLastCommitTime(path string) (string, error) {
	cmd := exec.Command("git", "log", "-1", `--format="%ad"`, "--", path)
	cmd.Dir = kepRepo
	t, err := cmd.Output()
	if err != nil {
		return "", errors.WithStack(err)
	}
	t = bytes.Trim(bytes.TrimSpace(t), `"`)
	lastUpdate, err := time.Parse("Mon Jan 2 15:04:05 2006 -0700", string(t))
	if err != nil {
		return "", errors.WithStack(err)
	}
	return lastUpdate.Format("2006-01-02"), nil
}

func FixData(path string, dryRun bool) error {
	_, head, meta, body, _ := openProposal(path)
	out := make(map[string]interface{})
	if err := yaml.Unmarshal(meta, out); err != nil {
		return errors.WithStack(err)
	}
	required := requiredKeys()

	// clean up bad casing of keys
	for key, value := range out {
		if strings.ToLower(key) == key {
			continue
		}
		out[strings.ToLower(key)] = value
		delete(out, key)
	}

	// keps/sig-network/0010-20180314-coredns-GA-proposal.md
	// figure out if the types are wrong
	for key, value := range out {
		if _, ok := required[key]; ok {
			required[key] = true
		}
		switch v := value.(type) {
		case string:
			switch key {
			case "replaces", "see-also", "superseded-by", "approvers", "reviewers", "participating-sigs", "authors":
				out[key] = interface{}([]string{v})
			case "creation-date":
				if _, err := time.Parse("2006-01-02", v); err != nil {
					s := timeRe.FindString(v)
					out[key] = s
					if s == "" {
						created, err := getCreatedTime(path)
						if err != nil {
							return errors.WithStack(err)
						}
						out[key] = created
					}
				}
			case "last-updated":
				if _, err := time.Parse("2006-01-02", v); err != nil {
					s := timeRe.FindString(v)
					out[key] = s
					if s == "" {
						lastUpdate, err := getLastCommitTime(path)
						if err != nil {
							return errors.WithStack(err)
						}
						out[key] = lastUpdate
					}
				}
			}
			continue
		case []interface{}:
			switch key {
			case "editors":
				// If they called it editors just pick the first one i guess
				out["editor"] = v[0]
			case "editor", "owning-sig", "title", "status":
				out[key] = "TBD"
				if len(v) > 0 {
					out[key] = v[0]
				}
			}
			continue
		case int:
			continue
		case float64:
			continue
		case nil:
			switch key {
			case "editor", "see-also", "participating-sigs", "replaces", "superseded-by":
				// all good with optional fields
				continue
			case "title":
				out[key] = "TBD"
			case "authors", "reviewers", "approvers":
				out[key] = interface{}([]string{"TBD"})
			case "creation-date":
				created, err := getCreatedTime(path)
				if err != nil {
					return errors.WithStack(err)
				}
				out[key] = created
			case "last-updated":
				lastUpdated, err := getLastCommitTime(path)
				if err != nil {
					return errors.WithStack(err)
				}
				out[key] = lastUpdated
			}
		default:
			fmt.Printf("UNKNOWN TYPE %T: ", v)
		}
	}

	// figure out if the key is simply missing
	for key, found := range required {
		if !found {
			switch key {
			case "title":
				out[key] = "TBD"
			case "authors", "reviewers", "approvers":
				out[key] = interface{}([]string{"TBD"})
			case "creation-date":
				created, err := getCreatedTime(path)
				if err != nil {
					return errors.WithStack(err)
				}
				out[key] = created
			case "last-updated":
				lastUpdated, err := getLastCommitTime(path)
				if err != nil {
					return errors.WithStack(err)
				}
				out[key] = lastUpdated
			}
		}
	}

	var buf bytes.Buffer

	order := []string{
		"title",
		"authors",
		"owning-sig",
		"participating-sigs",
		"reviewers",
		"approvers",
		"editor",
		"creation-date",
		"last-updated",
		"status",
		"see-also",
		"replaces",
		"superseded-by",
	}

	for _, o := range order {
		switch o {
		case "title", "creation-date", "last-updated", "editor", "owning-sig", "status":
			v, ok := out[o].(string)
			if !ok {
				// keep the field if it was there originally
				if bytes.Contains(meta, []byte(fmt.Sprintf("%s:", o))) {
					buf.WriteString(fmt.Sprintf("%s:\n", o))
				}
				continue
			}
			v = escapedValue(v, bytes.Contains(meta, []byte(fmt.Sprintf(`"%s`, v))))
			buf.WriteString(fmt.Sprintf("%s: %s\n", o, v))
		case "authors", "reviewers", "approvers", "see-also", "participating-sigs", "replaces", "superseded-by":
			items, ok := out[o].([]interface{})
			if !ok {
				if bytes.Contains(meta, []byte(fmt.Sprintf("%s:", o))) {
					buf.WriteString(fmt.Sprintf("%s:\n", o))
				}
				continue
			}
			if len(items) == 0 {
				// keep the field if it was there originally
				if bytes.Contains(meta, []byte(fmt.Sprintf("%s:", o))) {
					buf.WriteString(fmt.Sprintf("%s:\n", o))
				}
				continue
			}
			buf.WriteString(fmt.Sprintf("%s:\n", o))
			for _, item := range items {
				i, ok := item.(string)
				if !ok {
					continue
				}
				i = escapedValue(i, bytes.Contains(meta, []byte(fmt.Sprintf(`"%s`, i))))
				buf.WriteString(fmt.Sprintf("  - %v\n", i))
			}
		}
	}
	if err := writeFile(path, head, buf.Bytes(), body); err != nil {
		return errors.WithStack(err)
	}

	// marshal it into map[string]interface{}
	// we know what each key *SHOULD* look like
	// make it so
	return nil
}

func escapedValue(val string, originallyHas bool) string {
	if originallyHas {
		return fmt.Sprintf(`"%s"`, val)
	}
	if val[0] == '@' || val[0] == '[' || val[0] == '/' || strings.Contains(val, " -") || strings.Contains(val, ":") {
		return fmt.Sprintf(`"%s"`, val)
	}
	return val
}

var errRe = regexp.MustCompile(`line (\d+): cannot unmarshal !!map into string`)
var unexpectedHyphenRe = regexp.MustCompile(`line (\d+): did not find expected '-' indicator`)
var atsignRe = regexp.MustCompile(`line (\d+): found character that cannot start any token`)
var keyFindRe = regexp.MustCompile(`\s*[a-z]+:`)
var valStartsWithAmpersand = regexp.MustCompile(` (- )?"?@`)

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
		fmt.Println(match)
		lineNumber, err := strconv.Atoi(match[1])
		if err != nil {
			fmt.Printf("ERROR CONVERTING INT (2): %v\n", lineNumber)
		}
		fmt.Println(string(lines[lineNumber]))
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
	if err := cleanTrailingWhitespace(path, dryRun); err != nil {
		return err
	}
	if err := quoteUnquotedStringStartingWithAtSign(path, dryRun); err != nil {
		return err
	}

	if dryRun {
		return nil
	}

	// if we recognize the error then write out the good bytes followed by the
	// rest of the file
	return nil
}

func fixupStringListValuesToString(key, path string, dryRun bool) error {
	proposal, head, meta, body, err := openProposal(path)
	if err != nil {
		return err
	}
	if proposal == nil || proposal.Error == nil {
		return nil
	}
	lines := bytes.Split(meta, []byte("\n"))

	for _, line := range lines {
		if bytes.HasPrefix(bytes.TrimSpace(line), []byte(key)) {
			if bytes.HasSuffix(bytes.TrimSpace(line), []byte(":")) {

			}
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

// TODO: use this to ensure a required field, not editor
func ensureEditor(path string, dryRun bool) error {
	proposal, head, meta, body, err := openProposal(path)
	if err != nil {
		return err
	}
	if proposal == nil || proposal.Error == nil {
		return nil
	}
	lines := bytes.Split(meta, []byte("\n"))
	foundEditor := false
	for i, line := range lines {
		if bytes.Contains(line, []byte("editor:")) {
			foundEditor = true
		}
		if len(bytes.TrimSpace(line)) == 0 {
			lines[i] = []byte("editor: TBD\n")
			foundEditor = true
		}
	}

	if !foundEditor {
		lines = append(lines, []byte("editor: TBD\n"))
	}

	if dryRun {
		return nil
	}
	if err := writeFile(path, head, bytes.Join(lines, []byte("\n")), body); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func cleanTrailingWhitespace(path string, dryRun bool) error {
	proposal, head, meta, body, err := openProposal(path)
	if err != nil {
		return err
	}
	if proposal == nil || proposal.Error == nil {
		return nil
	}
	lines := bytes.Split(meta, []byte("\n"))

	for i, line := range lines {
		lines[i] = bytes.TrimRightFunc(line, unicode.IsSpace)
	}

	if dryRun {
		return nil
	}
	if err := writeFile(path, head, bytes.Join(lines, []byte("\n")), body); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func quoteUnquotedStringStartingWithAtSign(path string, dryRun bool) error {
	proposal, head, meta, body, err := openProposal(path)
	if err != nil {
		return err
	}
	if proposal == nil || proposal.Error == nil {
		return nil
	}
	lines := bytes.Split(meta, []byte("\n"))

	for i, line := range lines {
		if valStartsWithAmpersand.Match(line) {
			if bytes.Index(line, []byte(`"@`)) < 0 {
				lines[i] = bytes.Replace(line, []byte("@"), []byte(`"@`), 1)
			}
			if lines[i][len(lines[i])-1] != '"' {
				lines[i] = append(lines[i], '"')
			}
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
