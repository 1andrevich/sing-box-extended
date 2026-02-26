package option

type AdminPanelServiceOptions struct {
	ListenOptions
	Manager  string                    `json:"manager"`
	Database AdminPanelServiceDatabase `json:"database"`
	InboundTLSOptionsContainer
}

type AdminPanelServiceDatabase struct {
	Driver string `json:"driver"`
	DSN    string `json:"dsn"`
}
