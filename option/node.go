package option

type NodeServiceOptions struct {
	UUID               string
	Inbounds           []string `json:"inbounds"`
	ConnectionLimiters []string `json:"connection_limiters"`
	BandwidthLimiters  []string `json:"bandwidth_limiters"`
	Manager            string   `json:"manager"`
}
