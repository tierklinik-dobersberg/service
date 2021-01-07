package server

import (
	"github.com/ppacher/system-conf/conf"
)

// Listener defines a listener for the API server.
type Listener struct {
	Address        string
	TLSCertFile    string
	TLSKeyFile     string
	TrustedProxies []string
}

// ListenerSpec defines the available configuration values for the
// listener configuration sections.
var ListenerSpec = conf.SectionSpec{
	{
		Name:        "Address",
		Required:    true,
		Description: "Address to listen on in the format of <ip/hostname>:<port>.",
		Type:        conf.StringType,
	},
	{
		Name:        "CertificateFile",
		Description: "Path to the TLS certificate file (PEM format)",
		Type:        conf.StringType,
	},
	{
		Name:        "PrivateKeyFile",
		Description: "Path to the TLS private key file (PEM format)",
		Type:        conf.StringType,
	},
	{
		Name:        "TrustedProxies",
		Description: "IP addresses or CIDR subnet notation for trusted reverse proxies.",
		Type:        conf.StringSliceType,
	},
}
