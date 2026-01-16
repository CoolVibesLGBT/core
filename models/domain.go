package models

import "strings"

type DomainName string

const (
	CoolVibesLGBT DomainName = "coolvibes.lgbt"
	CoolVibesAPP  DomainName = "coolvibes.app"
	CoolVibesIO   DomainName = "coolvibes.io"

	KEWLSWAPCOM DomainName = "kewlswap.com"
	KEWLSWAPIO  DomainName = "kewlswap.io"
	KEWLSWAPAPP DomainName = "kewlswap.app"
)

type DomainKind string

const (
	CoolVibes DomainKind = "coolvibes"
	KewlSwap  DomainKind = "kewlswap"
)

var DomainMap = map[DomainName]DomainKind{
	// CoolVibes
	CoolVibesLGBT: CoolVibes,
	CoolVibesAPP:  CoolVibes,
	CoolVibesIO:   CoolVibes,

	// KewlSwap
	KEWLSWAPCOM: KewlSwap,
	KEWLSWAPIO:  KewlSwap,
	KEWLSWAPAPP: KewlSwap,
}

func HostToDomain(domain DomainName) (DomainKind, bool) {
	p, ok := DomainMap[domain]
	return p, ok
}

func ParseDomainName(s string) DomainKind {
	domainStr := DomainName(strings.ToLower(s))
	domain, ok := DomainMap[domainStr]
	if !ok {
		return CoolVibes
	}
	return domain
}
