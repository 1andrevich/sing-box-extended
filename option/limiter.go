package option

import (
	"github.com/sagernet/sing/common/byteformats"
)

type BandwidthLimiterOutboundOptions struct {
	Strategy       string                          `json:"strategy"`
	Mode           string                          `json:"mode"`
	ConnectionType string                          `json:"connection_type,omitempty"`
	Speed          *byteformats.NetworkBytesCompat `json:"speed"`
	Users          []BandwidthLimiterUser          `json:"users,omitempty"`
	Route          RouteOptions                    `json:"route"`
}

type BandwidthLimiterUser struct {
	Name           string                          `json:"name"`
	Strategy       string                          `json:"strategy"`
	Mode           string                          `json:"mode"`
	ConnectionType string                          `json:"connection_type,omitempty"`
	Speed          *byteformats.NetworkBytesCompat `json:"speed"`
}

type ConnectionLimiterOutboundOptions struct {
	Strategy       string                  `json:"strategy"`
	ConnectionType string                  `json:"connection_type,omitempty"`
	Count          uint32                  `json:"count"`
	Users          []ConnectionLimiterUser `json:"users,omitempty"`
	Route          RouteOptions            `json:"route"`
}

type ConnectionLimiterUser struct {
	Name           string `json:"name"`
	Strategy       string `json:"strategy"`
	ConnectionType string `json:"connection_type,omitempty"`
	Count          uint32 `json:"count"`
}
