package server

import (
	"net/http"

	"github.com/tierklinik-dobersberg/logger"
	"github.com/tierklinik-dobersberg/service/utils"
)

// WithTrustedProxyHeaders checks if the direct client (RemoteAddr field of req) is a trusted
// reverse proxy and if, extracts data from headers like X-Forwarded-For, ... and adds them
// to the request context.
// TODO(ppacher): should we allow to configure which headers are trusted? Or is it safe to
// assume that X-Forwarded-For, X-Forwarded-Proto, X-Forwarded-Host, X-Real-IP and the
// official Forwarded are fine to trust?
func WithTrustedProxyHeaders(proxies []string, req *http.Request) *http.Request {
	log := logger.From(req.Context())

	networks, err := utils.ParseNetworks(proxies)
	if err != nil {
		log.Errorf("failed to parse proxies: %s", err)
		return req
	}

	// check if we can trust the remote addr.
	if !networks.ContainsString(utils.RemovePort(req.RemoteAddr)) {
		return req
	}

	return utils.WithProxyHeaders(req)
}
