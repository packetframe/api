package util

import (
	"fmt"
	"net/smtp"
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
