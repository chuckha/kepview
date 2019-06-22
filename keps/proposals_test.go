package keps_test

import (
	"strings"
	"testing"

	"github.com/chuckha/kepctl/keps"
)

func TestValidParsing(t *testing.T) {
	testcases := []struct {
		name         string
		fileContents string
	}{
		{
			"simple test",
			`---
title: test
authors:
  - "@jpbetz"
  - "@roycaihw"
  - "@sttts"
owning-sig: sig-api-machinery
participating-sigs:
  - sig-api-machinery
  - sig-architecture
reviewers:
  - "@deads2k"
  - "@lavalamp"
  - "@liggitt"
  - "@mbohlool"
  - "@sttts"
approvers:
  - "@deads2k"
  - "@lavalamp"
creation-date: 2018-04-15
last-updated: 2018-04-24
status: provisional
---`,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			p := keps.NewParser()
			contents := strings.NewReader(tc.fileContents)
			out, err := p.Parse(contents)
			if err != nil {
				t.Fatalf("%+v", err)
			}
			if out == nil {
				t.Fatal("out should not be nil")
			}
		})
	}
}
