package util

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/smtp"
	"os"
	"strings"
)

// StrSliceContains runs a linear search over a string array
func StrSliceContains(array []string, element string) bool {
	for _, item := range array {
		if item == element {
			return true
		}
	}
	return false
}

// SendEmail sends an email
func SendEmail(host, user, pass, to, subject, body string) error {
	return smtp.SendMail(
		fmt.Sprintf("%s:%d", host, 587),
		smtp.PlainAuth("", user, pass, host),
		user,
		[]string{to},
		[]byte(fmt.Sprintf(`To: "%s" <%s>
From: "%s" <%s>
Subject: %s
%s`,
			to, to, user, user, subject, body,
		)),
	)
}

// SHA256File returns the SHA256 hash of a file
func SHA256File(file string) (string, error) {
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// SHA256 returns the SHA256 hash of a string
func SHA256(in string) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, strings.NewReader(in)); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
