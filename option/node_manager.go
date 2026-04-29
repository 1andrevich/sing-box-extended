package option

type NodeManagerServerServiceOptions struct {
	ListenOptions
	InboundTLSOptionsContainer
	Manager string `json:"manager"`
}

type NodeManagerClientServiceOptions struct {
	DialerOptions
	ServerOptions
	OutboundTLSOptionsContainer
}
