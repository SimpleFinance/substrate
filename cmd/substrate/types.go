package main

import (
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	"regexp"
	"strconv"
)

// EnvironmentDomainValue is the base domain name of a Substrate environment
type EnvironmentDomainValue string

var validEnvironmentDomainPattern = "^([a-z0-9]+(-[a-z0-9]+)*\\.)+[a-z]{2,}$"
var validEnvironmentDomainRegexp = regexp.MustCompile(validEnvironmentDomainPattern)

// Set checks whether an environment domain name is valid
func (name *EnvironmentDomainValue) Set(value string) error {
	if !validEnvironmentDomainRegexp.MatchString(value) {
		return fmt.Errorf(
			"invalid environment domain %v, must match %v",
			value,
			validEnvironmentDomainPattern)
	}
	*name = EnvironmentDomainValue(value)
	return nil
}

// String returns a the plain String value of an EnvironmentDomainValue
func (name *EnvironmentDomainValue) String() string {
	return string(*name)
}

// EnvironmentDomain sets a Kingpin Settings variable to be of type EnvironmentDomainValue
func EnvironmentDomain(s kingpin.Settings) (target *string) {
	target = new(string)
	s.SetValue((*EnvironmentDomainValue)(target))
	return
}

// EnvironmentNameValue is the name of a Substrate environment
type EnvironmentNameValue string

var validEnvironmentNamePattern = "^[a-z-]{3,10}$"
var validEnvironmentNameRegexp = regexp.MustCompile(validEnvironmentNamePattern)

// Set checks whether an environment name is valid
func (name *EnvironmentNameValue) Set(value string) error {
	if !validEnvironmentNameRegexp.MatchString(value) {
		return fmt.Errorf(
			"invalid environment name %v, must match %v",
			value,
			validEnvironmentNamePattern)
	}
	*name = EnvironmentNameValue(value)
	return nil
}

// String returns a the plain String value of an EnvironmentNameValue
func (name *EnvironmentNameValue) String() string {
	return string(*name)
}

// EnvironmentName sets a Kingpin Settings variable to be of type EnvironmentNameValue
func EnvironmentName(s kingpin.Settings) (target *string) {
	target = new(string)
	s.SetValue((*EnvironmentNameValue)(target))
	return
}

// EnvironmentIndexValue is the index of a Substrate environment
type EnvironmentIndexValue int

// Set checks whether an environment index is valid
func (i *EnvironmentIndexValue) Set(value string) error {
	idx, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("invalid environment index (%v)", err)
	}
	if idx < 0 || idx > 127 {
		return fmt.Errorf("environment index must be 0..127, not %d", idx)
	}
	*i = EnvironmentIndexValue(idx)
	return nil
}

// String returns the plain String value of an EnvironmentIndexValue
func (i *EnvironmentIndexValue) String() string {
	return fmt.Sprintf("%d", int(*i))
}

// EnvironmentIndex sets a Kingpin Settings variable to be of type EnvironmentIndexValue
func EnvironmentIndex(s kingpin.Settings) (target *int) {
	target = new(int)
	s.SetValue((*EnvironmentIndexValue)(target))
	return
}

// ZoneIndexValue is the index of a Substrate environment
type ZoneIndexValue int

// Set checks whether an environment index is valid
func (i *ZoneIndexValue) Set(value string) error {
	idx, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("invalid zone index (%v)", err)
	}
	if idx < 0 || idx > 15 {
		return fmt.Errorf("zone index must be 0..15, not %d", idx)
	}
	*i = ZoneIndexValue(idx)
	return nil
}

// String returns the plain String value of an ZoneIndexValue
func (i *ZoneIndexValue) String() string {
	return fmt.Sprintf("%d", int(*i))
}

// ZoneIndex sets a Kingpin Settings variable to be of type ZoneIndexValue
func ZoneIndex(s kingpin.Settings) (target *int) {
	target = new(int)
	s.SetValue((*ZoneIndexValue)(target))
	return
}
