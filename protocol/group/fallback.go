package group

import (
	"context"
	"net"
	"sync"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/outbound"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/logger"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"github.com/sagernet/sing/service"
)

func RegisterFallback(registry *outbound.Registry) {
	outbound.Register[option.FallbackOutboundOptions](registry, C.TypeFallback, NewFallback)
}

var _ adapter.OutboundGroup = (*Fallback)(nil)

type Fallback struct {
	outbound.Adapter
	ctx              context.Context
	outbound         adapter.OutboundManager
	logger           logger.ContextLogger
	tags             []string
	outbounds        map[string]adapter.Outbound
	lastUsedOutbound string

	mtx sync.Mutex
}

func NewFallback(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.FallbackOutboundOptions) (adapter.Outbound, error) {
	if len(options.Outbounds) == 0 {
		return nil, E.New("missing tags")
	}
	outbound := &Fallback{
		Adapter:          outbound.NewAdapter(C.TypeFallback, tag, []string{N.NetworkTCP, N.NetworkUDP}, options.Outbounds),
		ctx:              ctx,
		outbound:         service.FromContext[adapter.OutboundManager](ctx),
		logger:           logger,
		tags:             options.Outbounds,
		outbounds:        make(map[string]adapter.Outbound, len(options.Outbounds)),
		lastUsedOutbound: options.Outbounds[0],
	}
	return outbound, nil
}

func (s *Fallback) Start() error {
	for i, tag := range s.tags {
		outbound, loaded := s.outbound.Outbound(tag)
		if !loaded {
			return E.New("outbound ", i, " not found: ", tag)
		}
		s.outbounds[tag] = outbound
	}
	return nil
}

func (s *Fallback) Now() string {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	return s.lastUsedOutbound
}

func (s *Fallback) All() []string {
	return s.tags
}

func (s *Fallback) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	var conn net.Conn
	var err error
	for _, outbound := range s.outbounds {
		conn, err = outbound.DialContext(ctx, network, destination)
		if err != nil {
			s.logger.InfoContext(ctx, err)
			continue
		}
		s.mtx.Lock()
		defer s.mtx.Unlock()
		s.lastUsedOutbound = outbound.Tag()
		return conn, nil
	}
	return nil, err
}

func (s *Fallback) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	var conn net.PacketConn
	var err error
	for _, outbound := range s.outbounds {
		conn, err = outbound.ListenPacket(ctx, destination)
		if err != nil {
			s.logger.InfoContext(ctx, err)
			continue
		}
		s.mtx.Lock()
		defer s.mtx.Unlock()
		s.lastUsedOutbound = outbound.Tag()
		return conn, nil
	}
	return nil, err
}
