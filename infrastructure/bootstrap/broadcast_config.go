package bootstrap

import (
	infraBroadcast "core/infrastructure/broadcast"
	"os"
	"strings"
)

// broadcastGatewayConfigFromEnvironment is deliberately kept in the
// composition root so credentials never leak into handlers, use cases or
// workers.
func broadcastGatewayConfigFromEnvironment() infraBroadcast.Config {
	value := func(key string) string { return strings.TrimSpace(os.Getenv(key)) }
	return infraBroadcast.Config{
		Hornet: infraBroadcast.HornetConfig{
			BaseURL:         value("BROADCAST_HORNET_BASE_URL"),
			SessionToken:    value("BROADCAST_HORNET_SESSION_TOKEN"),
			ApplicationID:   value("BROADCAST_HORNET_APPLICATION_ID"),
			ClientUserAgent: value("BROADCAST_HORNET_CLIENT_USER_AGENT"),
			HTTPUserAgent:   value("BROADCAST_HORNET_HTTP_USER_AGENT"),
			Origin:          value("BROADCAST_HORNET_ORIGIN"),
			RefererBase:     value("BROADCAST_HORNET_REFERER_BASE"),
			NewRelic:        value("BROADCAST_HORNET_NEW_RELIC"),
			NewRelicID:      value("BROADCAST_HORNET_NEW_RELIC_ID"),
		},
		Growlr: infraBroadcast.GrowlrConfig{
			BaseURL:         value("BROADCAST_GROWLR_BASE_URL"),
			SessionToken:    value("BROADCAST_GROWLR_SESSION_TOKEN"),
			ApplicationID:   value("BROADCAST_GROWLR_APPLICATION_ID"),
			ClientKey:       value("BROADCAST_GROWLR_CLIENT_KEY"),
			InstallationID:  value("BROADCAST_GROWLR_INSTALLATION_ID"),
			OSVersion:       value("BROADCAST_GROWLR_OS_VERSION"),
			ClientVersion:   value("BROADCAST_GROWLR_CLIENT_VERSION"),
			ClientUserAgent: value("BROADCAST_GROWLR_CLIENT_USER_AGENT"),
			HTTPUserAgent:   value("BROADCAST_GROWLR_HTTP_USER_AGENT"),
			BuildVersion:    value("BROADCAST_GROWLR_BUILD_VERSION"),
			DisplayVersion:  value("BROADCAST_GROWLR_DISPLAY_VERSION"),
		},
	}
}
