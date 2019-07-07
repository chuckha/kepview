package main

import (
	"strings"
	"testing"
)

func TestMetadata(t *testing.T) {
	m := metadata{"h", "b", "c"}
	m.InsertLine("BOOM", 1)
	if strings.Join(m, " ") != "h BOOM b c" {
		t.Fatal(strings.Join(m, " "))
	}
}
func TestFieldKey(t *testing.T) {
	m := metadata{"test:", "  - abd", "my-test: hello", "deep:", "  - hello", "  - bye"}
	start, end := m.Field("my-test")
	if start != end {
		t.Fatal("start and end should be the same but were", start, end)
	}
	start, end = m.Field("test")
	if start != 0 {
		t.Fatal("start should have been 0 but was", start)
	}
	if end != 1 {
		t.Fatal("end should have been 1 but was", end)
	}
	start, end = m.Field("deep")
	if start != 3 {
		t.Fatal("start needs to be 3 ")
	}
	if end != 5 {
		t.Fatal("end needs to be 5 but was", end)
	}
}
