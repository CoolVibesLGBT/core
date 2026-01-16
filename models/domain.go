package models

import "strings"

type DomainName string

const (
	CoolVibesLGBT    DomainName = "coolvibes.lgbt"
	CoolVibesAPP     DomainName = "coolvibes.app"
	CoolVibesIO      DomainName = "coolvibes.io"
	CoolVibesLGBTAPI DomainName = "api.coolvibes.lgbt"
	CoolVibesAPPAPI  DomainName = "api.coolvibes.app"
	CoolVibesIOAPI   DomainName = "api.coolvibes.io"

	KEWLSWAPCOM    DomainName = "kewlswap.com"
	KEWLSWAPIO     DomainName = "kewlswap.io"
	KEWLSWAPAPP    DomainName = "kewlswap.app"
	APIKEWLSWAPCOM DomainName = "api.kewlswap.com"
	APIKEWLSWAPIO  DomainName = "api.kewlswap.io"
	APIKEWLSWAPAPP DomainName = "api.kewlswap.app"
)

type DomainKind string

const (
	CoolVibes DomainKind = "coolvibes"
	KewlSwap  DomainKind = "kewlswap"
)

var DomainMap = map[DomainName]DomainKind{
	// CoolVibes
	CoolVibesLGBT:    CoolVibes,
	CoolVibesAPP:     CoolVibes,
	CoolVibesIO:      CoolVibes,
	CoolVibesLGBTAPI: CoolVibes,
	CoolVibesAPPAPI:  CoolVibes,
	CoolVibesIOAPI:   CoolVibes,

	// KewlSwap
	KEWLSWAPCOM:    KewlSwap,
	KEWLSWAPIO:     KewlSwap,
	KEWLSWAPAPP:    KewlSwap,
	APIKEWLSWAPCOM: KewlSwap,
	APIKEWLSWAPIO:  KewlSwap,
	APIKEWLSWAPAPP: KewlSwap,
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
