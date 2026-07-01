package user

import (
	"errors"
	"net"
	"strconv"
	"strings"
)

type DomainKind string

const (
	CoolVibes     DomainKind = "coolvibes"
	KewlSwap      DomainKind = "kewlswap"
	UnknownDomain DomainKind = "unknown"
	AllDomains    DomainKind = "all"
)

var ErrInvalidDomain = errors.New("invalid domain")

var domainToKind = map[string]DomainKind{
	"coolvibes":          CoolVibes,
	"coolvibes.lgbt":     CoolVibes,
	"coolvibes.app":      CoolVibes,
	"coolvibes.io":       CoolVibes,
	"api.coolvibes.lgbt": CoolVibes,
	"api.coolvibes.app":  CoolVibes,
	"api.coolvibes.io":   CoolVibes,
	"192.168.0.14":       CoolVibes,
	"localhost":          CoolVibes,
	"127.0.0.1":          CoolVibes,
	"0.0.0.0":            CoolVibes,
	"::1":                CoolVibes,

	"kewl":             KewlSwap,
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

	if schemeIdx := strings.Index(input, "://"); schemeIdx >= 0 {
		input = input[schemeIdx+3:]
	}
	if slashIdx := strings.Index(input, "/"); slashIdx >= 0 {
		input = input[:slashIdx]
	}

	input = strings.ToLower(input)
	input = strings.TrimPrefix(input, "www.")
	if host, _, err := net.SplitHostPort(input); err == nil {
		input = strings.Trim(host, "[]")
	} else if colonIdx := strings.LastIndex(input, ":"); colonIdx > 0 && strings.Count(input, ":") == 1 {
		if _, err := strconv.Atoi(input[colonIdx+1:]); err == nil {
			input = input[:colonIdx]
		}
	}
	return input
}

func GetDomainKind(raw string) DomainKind {
	normalized := NormalizeDomain(raw)
	if kind, ok := domainToKind[normalized]; ok {
		return kind
	}
	return UnknownDomain
}

func ParseDomainKind(raw string) (DomainKind, error) {
	kind := GetDomainKind(raw)
	if !IsValidDomainByKind(kind) {
		return UnknownDomain, ErrInvalidDomain
	}
	return kind, nil
}

func IsValidDomain(raw string) bool {
	return GetDomainKind(raw) != UnknownDomain
}

func IsValidDomainByKind(domain DomainKind) bool {
	return domain != UnknownDomain
}
