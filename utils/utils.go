package utils

import (
	"net/url"
	"regexp"
	"strings"
)

func IsValidUrl(s string) bool {
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		return false
	}
	domainRegex := `^([a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}$`
	parsedUrl, err := url.ParseRequestURI(s)
	if err != nil {
		return false
	}
	matched, err := regexp.MatchString(domainRegex, parsedUrl.Host)
	if err != nil || !matched {
		return false
	}

	return true

}

func IsValidShortUrl(s string) bool {
	if len(s) < 3 || len(s) > 10 {
		return false
	}
	shortUrlRegex := `^[a-zA-Z0-9]+$`
	matched, err := regexp.MatchString(shortUrlRegex, s)
	if err != nil || !matched {
		return false
	}

	return true
}
