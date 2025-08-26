package util

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
)

func CheckName(name string, extraAllowedChars ...rune) error {
	if len(name) == 0 {
		return fmt.Errorf("empty names not allowed")
	}
	for _, c := range extraAllowedChars {
		name = strings.ReplaceAll(name, string(c), "")
	}
	errs := validation.IsDNS1123Label(name)
	if len(errs) != 0 {
		return fmt.Errorf("invalid name %s: %s", name, strings.Join(errs, ", "))
	}
	return nil
}
