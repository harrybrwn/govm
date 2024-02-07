package main

import (
	"errors"
	"regexp"
)

var semvarRe = regexp.MustCompile(`^[0-9]+\.[0-9]+(\.[0-9]+)?$`)

func validateVersion(v string) error {
	v = cleanVersionInput(v)
	if !semvarRe.MatchString(v) {
		return errors.New("bad version string")
	}
	return nil
}
