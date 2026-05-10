package bandwidth

import (
	"context"
	"net"
	"strconv"
	"sync"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/log"
	E "github.com/sagernet/sing/common/exceptions"
	"golang.org/x/time/rate"
)

type (
	CloseHandlerFunc  = func()
	ConnIDGetter      = func(context.Context, *adapter.InboundContext) (string, bool)
	ConnWrapper       = func(ctx context.Context, conn net.Conn, limiter Limiter, reverse bool) net.Conn
	PacketConnWrapper = func(ctx context.Context, conn net.PacketConn, limiter Limiter, reverse bool) net.PacketConn
)

type BandwidthStrategy interface {
	wrapConn(ctx context.Context, conn net.Conn, metadata *adapter.InboundContext, reverse bool) (net.Conn, error)
	wrapPacketConn(ctx context.Context, conn net.PacketConn, metadata *adapter.InboundContext, reverse bool) (net.PacketConn, error)
}

type BandwidthLimiterStrategy interface {
	getLimiter(ctx context.Context, metadata *adapter.InboundContext) (Limiter, CloseHandlerFunc, error)
}

type DefaultWrapStrategy struct {
	limiterStrategy   BandwidthLimiterStrategy
	connWrapper       ConnWrapper
	packetConnWrapper PacketConnWrapper
}

func NewDefaultWrapStrategy(limiterStrategy BandwidthLimiterStrategy, connWrapper ConnWrapper, packetConnWrapper PacketConnWrapper) *DefaultWrapStrategy {
	return &DefaultWrapStrategy{limiterStrategy, connWrapper, packetConnWrapper}
}

func (s *DefaultWrapStrategy) wrapConn(ctx context.Context, conn net.Conn, metadata *adapter.InboundContext, reverse bool) (net.Conn, error) {
	limiter, onClose, err := s.limiterStrategy.getLimiter(ctx, metadata)
	if err != nil {
		return nil, err
	}
	return NewConnWithCloseHandler(s.connWrapper(ctx, conn, limiter, reverse), onClose), nil
}

func (s *DefaultWrapStrategy) wrapPacketConn(ctx context.Context, conn net.PacketConn, metadata *adapter.InboundContext, reverse bool) (net.PacketConn, error) {
	limiter, onClose, err := s.limiterStrategy.getLimiter(ctx, metadata)
	if err != nil {
		return nil, err
	}
	return NewPacketConnWithCloseHandler(s.packetConnWrapper(ctx, conn, limiter, reverse), onClose), nil
}

type GlobalBandwidthStrategy struct {
	limiter Limiter
}

func NewGlobalBandwidthStrategy(speed uint64, flowKeys []string) (*GlobalBandwidthStrategy, error) {
	limiter, err := createSpeedLimiter(speed, flowKeys)
	if err != nil {
		return nil, err
	}
	return &GlobalBandwidthStrategy{
		limiter: limiter,
	}, nil
}

func (s *GlobalBandwidthStrategy) getLimiter(ctx context.Context, metadata *adapter.InboundContext) (Limiter, CloseHandlerFunc, error) {
	return s.limiter, func() {}, nil
}

type idBandwidthLimiter struct {
	limiter Limiter
	handles uint32
}

type ConnectionBandwidthStrategy struct {
	limiters     map[string]*idBandwidthLimiter
	connIDGetter ConnIDGetter
	speed        uint64
	flowKeys     []string
	mtx          sync.Mutex
}

func NewConnectionBandwidthStrategy(connIDGetter ConnIDGetter, speed uint64, flowKeys []string) *ConnectionBandwidthStrategy {
	return &ConnectionBandwidthStrategy{
		limiters:     make(map[string]*idBandwidthLimiter),
		connIDGetter: connIDGetter,
		speed:        speed,
		flowKeys:     flowKeys,
	}
}

func (s *ConnectionBandwidthStrategy) getLimiter(ctx context.Context, metadata *adapter.InboundContext) (Limiter, CloseHandlerFunc, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	id, ok := s.connIDGetter(ctx, metadata)
	if !ok {
		return nil, nil, E.New("id not found")
	}
	limiter, ok := s.limiters[id]
	if !ok {
		newLimiter, err := createSpeedLimiter(s.speed, s.flowKeys)
		if err != nil {
			return nil, nil, err
		}
		limiter = &idBandwidthLimiter{
			limiter: newLimiter,
		}
		s.limiters[id] = limiter
	}
	limiter.handles++
	var once sync.Once
	return limiter.limiter, func() {
		once.Do(func() {
			s.mtx.Lock()
			defer s.mtx.Unlock()
			limiter.handles--
			if limiter.handles == 0 {
				delete(s.limiters, id)
			}
		})
	}, nil
}

type UsersBandwidthStrategy struct {
	strategies map[string]BandwidthStrategy
	mtx        sync.Mutex
}

func NewUsersBandwidthStrategy(strategies map[string]BandwidthStrategy) *UsersBandwidthStrategy {
	return &UsersBandwidthStrategy{
		strategies: strategies,
	}
}

func (s *UsersBandwidthStrategy) wrapConn(ctx context.Context, conn net.Conn, metadata *adapter.InboundContext, reverse bool) (net.Conn, error) {
	strategy, err := s.getStrategy(ctx, metadata)
	if err != nil {
		return nil, err
	}
	return strategy.wrapConn(ctx, conn, metadata, reverse)
}

func (s *UsersBandwidthStrategy) wrapPacketConn(ctx context.Context, conn net.PacketConn, metadata *adapter.InboundContext, reverse bool) (net.PacketConn, error) {
	strategy, err := s.getStrategy(ctx, metadata)
	if err != nil {
		return nil, err
	}
	return strategy.wrapPacketConn(ctx, conn, metadata, reverse)
}

func (s *UsersBandwidthStrategy) getStrategy(ctx context.Context, metadata *adapter.InboundContext) (BandwidthStrategy, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	var user string
	if metadata != nil {
		user = metadata.User
	}
	strategy, ok := s.strategies[user]
	if ok {
		return strategy, nil
	}
	return nil, E.New("user strategy not found: ", user)
}

type ManagerBandwidthStrategy struct {
	*UsersBandwidthStrategy
}

func NewManagerBandwidthStrategy() *ManagerBandwidthStrategy {
	return &ManagerBandwidthStrategy{
		UsersBandwidthStrategy: NewUsersBandwidthStrategy(map[string]BandwidthStrategy{}),
	}
}

func (s *ManagerBandwidthStrategy) UpdateStrategies(strategies map[string]BandwidthStrategy) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.strategies = strategies
}

type BypassBandwidthStrategy struct{}

func NewBypassBandwidthStrategy() *BypassBandwidthStrategy {
	return &BypassBandwidthStrategy{}
}

func (s *BypassBandwidthStrategy) wrapConn(ctx context.Context, conn net.Conn, metadata *adapter.InboundContext, reverse bool) (net.Conn, error) {
	return conn, nil
}

func (s *BypassBandwidthStrategy) wrapPacketConn(ctx context.Context, conn net.PacketConn, metadata *adapter.InboundContext, reverse bool) (net.PacketConn, error) {
	return conn, nil
}

func CreateStrategy(strategy string, mode string, connectionType string, speed uint64, flowKeys []string) (BandwidthStrategy, error) {
	var limiterStrategy BandwidthLimiterStrategy
	switch strategy {
	case "global":
		var err error
		limiterStrategy, err = NewGlobalBandwidthStrategy(speed, flowKeys)
		if err != nil {
			return nil, err
		}
	case "connection":
		var connIDGetter ConnIDGetter
		switch connectionType {
		case "hwid":
			connIDGetter = func(ctx context.Context, metadata *adapter.InboundContext) (string, bool) {
				id, ok := ctx.Value("hwid").(string)
				return id, ok
			}
		case "mux":
			connIDGetter = func(ctx context.Context, metadata *adapter.InboundContext) (string, bool) {
				id, ok := log.MuxIDFromContext(ctx)
				if !ok {
					return "", ok
				}
				return strconv.FormatUint(uint64(id.ID), 10), ok
			}
		case "source_ip":
			connIDGetter = func(ctx context.Context, metadata *adapter.InboundContext) (string, bool) {
				return metadata.Source.IPAddr().String(), true
			}
		case "default", "":
			connIDGetter = func(ctx context.Context, metadata *adapter.InboundContext) (string, bool) {
				id, ok := log.IDFromContext(ctx)
				if !ok {
					return "", ok
				}
				return strconv.FormatUint(uint64(id.ID), 10), ok
			}
		default:
			return nil, E.New("connection type not found: ", connectionType)
		}
		limiterStrategy = NewConnectionBandwidthStrategy(connIDGetter, speed, flowKeys)
	case "bypass":
		return NewBypassBandwidthStrategy(), nil
	default:
		return nil, E.New("strategy not found: ", strategy)
	}
	var (
		connWrapper       ConnWrapper
		packetConnWrapper PacketConnWrapper
	)
	switch mode {
	case "download":
		connWrapper = connWithDownloadBandwidthWrapper
		packetConnWrapper = packetConnWithDownloadBandwidthWrapper
	case "upload":
		connWrapper = connWithUploadBandwidthWrapper
		packetConnWrapper = packetConnWithUploadBandwidthWrapper
	case "bidirectional":
		connWrapper = connWithBidirectionalBandwidthWrapper
		packetConnWrapper = packetConnWithBidirectionalBandwidthWrapper
	default:
		return nil, E.New("mode not found: ", mode)
	}
	return NewDefaultWrapStrategy(limiterStrategy, connWrapper, packetConnWrapper), nil
}

func createSpeedLimiter(speed uint64, flowKeys []string) (Limiter, error) {
	var limiter Limiter = rate.NewLimiter(rate.Limit(float64(speed)), 65536)
	for i := len(flowKeys) - 1; i >= 0; i-- {
		getter, err := flowKeysConnIDGetter(flowKeys[i])
		if err != nil {
			return nil, err
		}
		limiter = NewFlowKeysLimiter(getter, limiter)
	}
	return limiter, nil
}

func flowKeysConnIDGetter(name string) (ConnIDGetter, error) {
	switch name {
	case "user":
		return func(ctx context.Context, metadata *adapter.InboundContext) (string, bool) {
			return metadata.User, true
		}, nil
	case "destination":
		return func(ctx context.Context, metadata *adapter.InboundContext) (string, bool) {
			return metadata.Destination.String(), true
		}, nil
	case "source_ip":
		return func(ctx context.Context, metadata *adapter.InboundContext) (string, bool) {
			return metadata.Source.IPAddr().String(), true
		}, nil
	case "hwid":
		return func(ctx context.Context, metadata *adapter.InboundContext) (string, bool) {
			id, ok := ctx.Value("hwid").(string)
			return id, ok
		}, nil
	case "mux":
		return func(ctx context.Context, metadata *adapter.InboundContext) (string, bool) {
			id, ok := log.MuxIDFromContext(ctx)
			if !ok {
				return "", ok
			}
			return strconv.FormatUint(uint64(id.ID), 10), ok
		}, nil
	default:
		return nil, E.New("flow key not found: ", name)
	}
}
