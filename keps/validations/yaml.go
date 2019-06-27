package validations

import (
	"fmt"
	"strings"
)

type KeyMustBeString struct {
	key interface{}
}

func (k *KeyMustBeString) Error() string {
	return fmt.Sprintf("key %[1]v must be a string but it is a %[1]T", k.key)
}

type ValueMustBeString struct {
	key   string
	value interface{}
}

func (v *ValueMustBeString) Error() string {
	return fmt.Sprintf("%q must be a string but it is a %T: %v", v.key, v.value, v.value)
}

type ValueMustBeListOfStrings struct {
	key   string
	value interface{}
}

func (v *ValueMustBeListOfStrings) Error() string {
	return fmt.Sprintf("%q must be a list of strings: %v", v.key, v.value)
}

type MustHaveOneValue struct {
	key string
}

func (m *MustHaveOneValue) Error() string {
	return fmt.Sprintf("%q must have a value", m.key)
}

type MustHaveAtLeastOneValue struct {
	key string
}

func (m *MustHaveAtLeastOneValue) Error() string {
	return fmt.Sprintf("%q must have at least one value", m.key)
}
func ValidateStructure(parsed map[interface{}]interface{}) error {
	for key, value := range parsed {
		// First off the key has to be a string. fact.
		k, ok := key.(string)
		if !ok {
			return &KeyMustBeString{k}
		}
		empty := value == nil

		// figure out the types
		switch strings.ToLower(k) {
		case "title", "owning-sig", "editor", "status", "creation-date", "last-updated":
			v, ok := value.(string)
			if !ok && v == "" {
				return &MustHaveOneValue{k}
			}
			if !ok {
				return &ValueMustBeString{k, v}
			}
		// These are optional lists, so skip if there is no value
		case "participating-sigs", "replaces", "superseded-by", "see-also":
			if empty {
				continue
			}
			fallthrough
		case "authors", "reviewers", "approvers":
			v, ok := value.([]interface{})
			if !ok && len(v) == 0 {
				return &MustHaveAtLeastOneValue{k}
			}
			if !ok {
				return &ValueMustBeListOfStrings{k, v}
			}
		}
	}
	return nil
}
