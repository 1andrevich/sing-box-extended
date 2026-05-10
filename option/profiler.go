package option

import "github.com/sagernet/sing/common/json/badoption"

type ProfilerServiceOptions struct {
	Listen       string             `json:"listen,omitempty"`
	ReadTimeout  badoption.Duration `json:"read_timeout,omitempty"`
	WriteTimeout badoption.Duration `json:"write_timeout,omitempty"`
}
