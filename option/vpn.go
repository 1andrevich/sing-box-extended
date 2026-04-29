package option

import (
	"net/netip"

	"github.com/sagernet/sing/common/json/badoption"
)

type VPNClientEndpointOptions struct {
	Address  netip.Addr `json:"address"`
	Key      string     `json:"key"`
	Outbound Outbound   `json:"outbound"`
}

type VPNServerEndpointOptions struct {
	Address        netip.Addr         `json:"address"`
	Users          []VPNUser          `json:"users"`
	Inbounds       []Inbound          `json:"inbounds"`
	ConnectTimeout badoption.Duration `json:"connect_timeout,omitempty"`
}

type VPNUser struct {
	Address netip.Addr `json:"address"`
	Key     string     `json:"key"`
}
