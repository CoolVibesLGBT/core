package models

import (
	"net/url"
	"strings"
)

type DomainKind string

const (
	CoolVibes     DomainKind = "coolvibes"
	KewlSwap      DomainKind = "kewlswap"
	UnknownDomain DomainKind = "unknown"
	AllDomains    DomainKind = "all"
)

var domainToKind = map[string]DomainKind{
	// CoolVibes
	"coolvibes.lgbt":     CoolVibes,
	"coolvibes.app":      CoolVibes,
	"coolvibes.io":       CoolVibes,
	"api.coolvibes.lgbt": CoolVibes,
	"api.coolvibes.app":  CoolVibes,
	"api.coolvibes.io":   CoolVibes,

	// KewlSwap
	"kewlswap.com":     KewlSwap,
	"kewlswap.io":      KewlSwap,
	"kewlswap.app":     KewlSwap,
	"api.kewlswap.com": KewlSwap,
	"api.kewlswap.io":  KewlSwap,
	"api.kewlswap.app": KewlSwap,
}

func NormalizeDomain(input string) string {
	input = strings.TrimSpace(input)
	input = strings.TrimRight(input, "/")
	u, err := url.Parse(input)
	if err == nil && u.Host != "" {
		input = u.Host
	} else {
		u, err = url.Parse("https://" + input)
		if err == nil {
			input = u.Host
		}
	}

	input = strings.ToLower(input)
	input = strings.TrimPrefix(input, "www.")
	return input
}

func GetDomainKind(raw string) DomainKind {
	normalized := NormalizeDomain(raw)
	if kind, ok := domainToKind[normalized]; ok {
		return kind
	}
	return UnknownDomain
}

// IsValidDomain returns true if domain belongs to any known domain groups
func IsValidDomain(raw string) bool {
	return GetDomainKind(raw) != UnknownDomain
}

func IsValidDomainByKind(domain DomainKind) bool {
	return domain != UnknownDomain
}
