package utils

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/apex/log"
	"github.com/tierklinik-dobersberg/logger"
)

type contextKey string

// Context keys used to add various proxy headers to a
// request context.
const (
	XRealIPHeaderKey   = contextKey("http:x-real-ip")
	XForwardedForKey   = contextKey("http:x-forwarded-for")
	XForwardedProtoKey = contextKey("http:x-forwarded-proto")
	XForwardedHostKey  = contextKey("http:x-forwarded-host")
)

// RealClientIP returns the real IP address of the client that
// iniated req. RealClientIP returns the IP address form any
// forwarded proxy header set by WithTrustedProxyHeaders.
// If none is present the RemoteAddr field of req is parsed
// an returned. In case of an error, nil is returne.d
func RealClientIP(req *http.Request) net.IP {
	if val, _ := req.Context().Value(XForwardedForKey).(net.IP); val != nil {
		return val
	}

	if val, _ := req.Context().Value(XRealIPHeaderKey).(net.IP); val != nil {
		return val
	}

	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return nil
	}

	return net.ParseIP(host)
}

// WithProxyHeaders parses all X-Forwarded-, X-Real-IP and
// Forwarded headers and adds their values to the request
// context. Better not use directly as the server package
// already calls WithProxyHeaders but guarded in trusted-proxy
// checks.
func WithProxyHeaders(req *http.Request) *http.Request {
	ph := newProxyHeaders(req)

	ph.addXRealIP(req.Header).
		addXForwardedFor(req.Header).
		addXForwardedHost(req.Header).
		addXForwardedProto(req.Header).
		parseForwarded(req.Header)

	ctx := req.Context()
	for key, value := range ph.values {
		ctx = context.WithValue(ctx, key, value)
	}

	return req.WithContext(ctx)
}

type proxyHeaders struct {
	req    *http.Request
	log    logger.Logger
	values map[interface{}]interface{}
}

func newProxyHeaders(req *http.Request) *proxyHeaders {
	return &proxyHeaders{
		req:    req,
		log:    logger.From(req.Context()),
		values: make(map[interface{}]interface{}),
	}
}

func (ph *proxyHeaders) addXRealIP(h http.Header) *proxyHeaders {
	val := h.Get("X-Real-IP")
	if val == "" {
		return ph
	}

	if realIP := ParseIP(val); realIP != nil {
		ph.values[XRealIPHeaderKey] = realIP
	} else {
		log.Errorf("failed to parse header for X-Real-IP: %q", val)
	}
	return ph
}

func (ph *proxyHeaders) addXForwardedFor(h http.Header) *proxyHeaders {
	if val := h.Get("X-Forwarded-For"); val != "" {
		parts := strings.Split(val, ",")
		if len(parts) >= 1 {
			if realIP := ParseIP(parts[0]); realIP != nil {
				ph.values[XForwardedForKey] = realIP
			} else {
				log.Errorf("failed to parse X-Forwarded-For header %q", val)
			}
		} else {
			log.Errorf("invalid X-Forwarded-For header %q", val)
		}
	}
	return ph
}

func (ph *proxyHeaders) addXForwardedProto(h http.Header) *proxyHeaders {
	if val := h.Get("X-Forwarded-Proto"); val != "" {
		ph.values[XForwardedProtoKey] = val
	}
	return ph
}

func (ph *proxyHeaders) addXForwardedHost(h http.Header) *proxyHeaders {
	if val := h.Get("X-Forwarded-Host"); val != "" {
		ph.values[XForwardedHostKey] = val
	}
	return ph
}

func (ph *proxyHeaders) parseForwarded(h http.Header) *proxyHeaders {
	hasForwardedFor := false
	if forwardedHeaders := h.Values("Forwarded"); len(forwardedHeaders) > 0 {
		for _, forwarded := range forwardedHeaders {
			for _, tokenPairs := range strings.Split(forwarded, ";") {
				for _, value := range strings.Split(tokenPairs, ",") {
					value = strings.TrimSpace(value)
					tokens := strings.SplitN(value, "=", 2)
					if len(tokens) != 2 {
						log.Errorf("invalid number of tokens in forwarded for header %q: %q", forwarded, value)
						continue
					}

					switch strings.ToLower(tokens[0]) {
					case "by":
						// ignored
					case "host":
						ph.values[XForwardedHostKey] = tokens[1]

					case "proto":
						ph.values[XForwardedProtoKey] = tokens[1]

					case "for":
						ipStr := RemovePort(tokens[1])
						if realIP := ParseIP(ipStr); realIP != nil {
							if !hasForwardedFor {
								ph.values[XForwardedForKey] = realIP
								hasForwardedFor = true
							} else {
								log.Infof("found additional for=%s", realIP.String())
							}
						} else {
							log.Errorf("failed to parse IP from for=%s: (extracted-ip: %s)", tokens[1], ipStr)
						}
					}
				}
			}
		}
	}
	return ph
}

// RemovePort tries to strip any :<port> suffix from str.
// It does not validate the rest of str in any other way.
func RemovePort(str string) string {
	for idx := len(str) - 1; idx >= 0; idx-- {
		if str[idx] == ':' {
			return str[0:idx]
		}

		if str[idx] == ']' {
			return str
		}
	}

	return str
}
