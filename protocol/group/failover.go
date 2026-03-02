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

func RegisterFailover(registry *outbound.Registry) {
	outbound.Register[option.FailoverOutboundOptions](registry, C.TypeFailover, NewFailover)
}

var (
	_ adapter.OutboundGroup = (*Failover)(nil)
)

type Failover struct {
	outbound.Adapter
	ctx              context.Context
	outbound         adapter.OutboundManager
	logger           logger.ContextLogger
	tags             []string
	outbounds        map[string]adapter.Outbound
	lastUsedOutbound string

	mtx sync.Mutex
}

func NewFailover(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.FailoverOutboundOptions) (adapter.Outbound, error) {
	if len(options.Outbounds) == 0 {
		return nil, E.New("missing tags")
	}
	outbound := &Failover{
		Adapter:          outbound.NewAdapter(C.TypeFailover, tag, []string{N.NetworkTCP, N.NetworkUDP}, options.Outbounds),
		ctx:              ctx,
		outbound:         service.FromContext[adapter.OutboundManager](ctx),
		logger:           logger,
		tags:             options.Outbounds,
		outbounds:        make(map[string]adapter.Outbound, len(options.Outbounds)),
		lastUsedOutbound: options.Outbounds[0],
	}
	return outbound, nil
}

func (s *Failover) Start() error {
	for i, tag := range s.tags {
		outbound, loaded := s.outbound.Outbound(tag)
		if !loaded {
			return E.New("outbound ", i, " not found: ", tag)
		}
		s.outbounds[tag] = outbound
	}
	return nil
}

func (s *Failover) Now() string {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	return s.lastUsedOutbound
}

func (s *Failover) All() []string {
	return s.tags
}

func (s *Failover) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	var conn net.Conn
	var err error
	for _, outbound := range s.outbounds {
		conn, err = outbound.DialContext(ctx, network, destination)
		if err != nil {
			s.logger.ErrorContext(ctx, err)
			continue
		}
		s.mtx.Lock()
		defer s.mtx.Unlock()
		s.lastUsedOutbound = outbound.Tag()
		return conn, nil
	}
	return nil, err
}

func (s *Failover) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	var conn net.PacketConn
	var err error
	for _, outbound := range s.outbounds {
		conn, err = outbound.ListenPacket(ctx, destination)
		if err != nil {
			s.logger.ErrorContext(ctx, err)
			continue
		}
		return conn, nil
	}
	return nil, err
}
