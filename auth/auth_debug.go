// +build debug

package main

import (
	"net/http"
	"time"
)

func sessionCookie(value string) *http.Cookie {
	return &http.Cookie{
		Name:     "SessionID",
		Value:    value,
		HttpOnly: true,
		Expires:  time.Now().AddDate(0, 1, 0), // 1 month expiry
	}
}
