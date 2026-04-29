package option

type FailoverInboundOptions struct {
	Inbounds []Inbound `json:"inbounds"`
}

type FailoverOutboundOptions struct {
	Outbounds []Outbound `json:"outbounds"`
}
