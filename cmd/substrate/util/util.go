package util

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os/user"
	"strings"
)

// Confirm prints out a prompt on stdout and reads a y/n response from the
// console, returning nil on "yes" or an error if "no".
func Confirm(prompt string) error {
	fmt.Printf("\n%s [y/n]: ", prompt)

	var confirm string
	_, err := fmt.Scanln(&confirm)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(strings.ToLower(confirm), "y") {
		return fmt.Errorf("canceling")
	}

	return nil
}

// CurrentUser returns the username of the current user, or "" on error
func CurrentUser() string {
	user, err := user.Current()
	if err != nil {
		return ""
	}
	return user.Username
}

// RandomHex generates numBytes bytes of random data and returns it as a hex encoded string. This function can panic if a source of random data is unavailable.
func RandomHex(numBytes int) string {
	bytes := make([]byte, numBytes)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}

// StringSlicesEqual tests if two slices of strings are equal
func StringSlicesEqual(a []string, b []string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
