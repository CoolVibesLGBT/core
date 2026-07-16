package models

import (
	domainuser "core/domain/user"
)

type DomainKind = domainuser.DomainKind

const (
	CoolVibes     = domainuser.CoolVibes
	KewlSwap      = domainuser.KewlSwap
	UnknownDomain = domainuser.UnknownDomain
	AllDomains    = domainuser.AllDomains
)

func NormalizeDomain(input string) string {
	return domainuser.NormalizeDomain(input)
}

func GetDomainKind(raw string) DomainKind {
	return domainuser.GetDomainKind(raw)
}

func IsValidDomain(raw string) bool {
	return domainuser.IsValidDomain(raw)
}

func IsValidDomainByKind(domain DomainKind) bool {
	return domainuser.IsValidDomainByKind(domain)
}
