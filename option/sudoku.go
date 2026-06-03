package option

import "github.com/sagernet/sing/common/json/badoption"

type SudokuOutboundOptions struct {
	DialerOptions
	ServerOptions
	Key                string            `json:"key"`
	AEADMethod         string            `json:"aead_method,omitempty"`
	PaddingMin         *int              `json:"padding_min,omitempty"`
	PaddingMax         *int              `json:"padding_max,omitempty"`
	TableType          string            `json:"table_type,omitempty"`
	EnablePureDownlink *bool             `json:"enable_pure_downlink,omitempty"`
	CustomTable        string            `json:"custom_table,omitempty"`
	CustomTables       []string          `json:"custom_tables,omitempty"`
	HTTPMask           *SudokuHTTPMask   `json:"http_mask,omitempty"`
}

type SudokuHTTPMask struct {
	Enabled   bool                      `json:"enabled,omitempty"`
	Mode      string                    `json:"mode,omitempty"`
	Host      string                    `json:"host,omitempty"`
	PathRoot  string                    `json:"path_root,omitempty"`
	Multiplex string                    `json:"multiplex,omitempty"`
	TLS       *SudokuOutboundTLSOptions `json:"tls,omitempty"`
}

type SudokuInboundOptions struct {
	ListenOptions
	Key                string   `json:"key"`
	AEADMethod         string   `json:"aead_method,omitempty"`
	PaddingMin         *int     `json:"padding_min,omitempty"`
	PaddingMax         *int     `json:"padding_max,omitempty"`
	TableType          string   `json:"table_type,omitempty"`
	HandshakeTimeout   *int     `json:"handshake_timeout,omitempty"`
	EnablePureDownlink *bool    `json:"enable_pure_downlink,omitempty"`
	CustomTable        string   `json:"custom_table,omitempty"`
	CustomTables       []string `json:"custom_tables,omitempty"`
	DisableHTTPMask    bool     `json:"disable_http_mask,omitempty"`
	HTTPMaskMode       string   `json:"http_mask_mode,omitempty"`
	PathRoot           string   `json:"path_root,omitempty"`
	Fallback           string   `json:"fallback,omitempty"`
}

type SudokuOutboundTLSOptions struct {
	Enabled               bool                                `json:"enabled,omitempty"`
	Fragment              bool                                `json:"fragment,omitempty"`
	FragmentFallbackDelay badoption.Duration                  `json:"fragment_fallback_delay,omitempty"`
	RecordFragment        bool                                `json:"record_fragment,omitempty"`
	KernelTx              bool                                `json:"kernel_tx,omitempty"`
	KernelRx              bool                                `json:"kernel_rx,omitempty"`
}


