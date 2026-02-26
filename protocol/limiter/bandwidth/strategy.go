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
	ConnWrapper       = func(ctx context.Context, conn net.Conn, limiter *rate.Limiter, reverse bool) net.Conn
	PacketConnWrapper = func(ctx context.Context, conn net.PacketConn, limiter *rate.Limiter, reverse bool) net.PacketConn
)

type BandwidthStrategy interface {
	wrapConn(ctx context.Context, conn net.Conn, metadata *adapter.InboundContext, reverse bool) (net.Conn, error)
	wrapPacketConn(ctx context.Context, conn net.PacketConn, metadata *adapter.InboundContext, reverse bool) (net.PacketConn, error)
}

type BandwidthLimiterStrategy interface {
	getLimiter(ctx context.Context, metadata *adapter.InboundContext) (*rate.Limiter, CloseHandlerFunc, error)
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
	limiter *rate.Limiter
}

func NewGlobalBandwidthStrategy(speed uint64) *GlobalBandwidthStrategy {
	return &GlobalBandwidthStrategy{
		limiter: createSpeedLimiter(speed),
	}
}

func (s *GlobalBandwidthStrategy) getLimiter(ctx context.Context, metadata *adapter.InboundContext) (*rate.Limiter, CloseHandlerFunc, error) {
	return s.limiter, func() {}, nil
}

type idBandwidthLimiter struct {
	limiter *rate.Limiter
	handles uint32
}

type ConnectionBandwidthStrategy struct {
	limiters     map[string]*idBandwidthLimiter
	connIDGetter ConnIDGetter
	speed        uint64
	mtx          sync.Mutex
}

func NewConnectionBandwidthStrategy(connIDGetter ConnIDGetter, speed uint64) *ConnectionBandwidthStrategy {
	return &ConnectionBandwidthStrategy{
		limiters:     make(map[string]*idBandwidthLimiter),
		connIDGetter: connIDGetter,
		speed:        speed,
	}
}

func (s *ConnectionBandwidthStrategy) getLimiter(ctx context.Context, metadata *adapter.InboundContext) (*rate.Limiter, CloseHandlerFunc, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	id, ok := s.connIDGetter(ctx, metadata)
	if !ok {
		return nil, nil, E.New("id not found")
	}
	limiter, ok := s.limiters[id]
	if !ok {
		limiter = &idBandwidthLimiter{
			limiter: createSpeedLimiter(s.speed),
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

func CreateStrategy(strategy string, mode string, connectionType string, speed uint64) (BandwidthStrategy, error) {
	var limiterStrategy BandwidthLimiterStrategy
	switch strategy {
	case "global":
		limiterStrategy = NewGlobalBandwidthStrategy(speed)
	case "connection":
		var connIDGetter ConnIDGetter
		switch connectionType {
		case "mux":
			connIDGetter = func(ctx context.Context, metadata *adapter.InboundContext) (string, bool) {
				id, ok := log.MuxIDFromContext(ctx)
				if !ok {
					return "", ok
				}
				return strconv.FormatUint(uint64(id.ID), 10), ok
			}
		case "hwid":
			connIDGetter = func(ctx context.Context, metadata *adapter.InboundContext) (string, bool) {
				id, ok := ctx.Value("hwid").(string)
				return id, ok
			}
		case "ip":
			connIDGetter = func(ctx context.Context, metadata *adapter.InboundContext) (string, bool) {
				return metadata.Source.IPAddr().String(), true
			}
		default:
			return nil, E.New("connection type not found: ", connectionType)
		}
		limiterStrategy = NewConnectionBandwidthStrategy(connIDGetter, speed)
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
	case "duplex":
		connWrapper = connWithDuplexBandwidthWrapper
		packetConnWrapper = packetConnWithDuplexBandwidthWrapper
	default:
		return nil, E.New("mode not found: ", mode)
	}
	return NewDefaultWrapStrategy(limiterStrategy, connWrapper, packetConnWrapper), nil
}

func createSpeedLimiter(speed uint64) *rate.Limiter {
	return rate.NewLimiter(rate.Limit(float64(speed)), 10000)
}

func connWithDownloadBandwidthWrapper(ctx context.Context, conn net.Conn, limiter *rate.Limiter, reverse bool) net.Conn {
	if reverse {
		return NewConnWithUploadBandwidthLimiter(ctx, conn, limiter)
	}
	return NewConnWithDownloadBandwidthLimiter(ctx, conn, limiter)
}

func connWithUploadBandwidthWrapper(ctx context.Context, conn net.Conn, limiter *rate.Limiter, reverse bool) net.Conn {
	if reverse {
		return NewConnWithDownloadBandwidthLimiter(ctx, conn, limiter)
	}
	return NewConnWithUploadBandwidthLimiter(ctx, conn, limiter)
}

func connWithDuplexBandwidthWrapper(ctx context.Context, conn net.Conn, limiter *rate.Limiter, reverse bool) net.Conn {
	return NewConnWithUploadBandwidthLimiter(ctx, NewConnWithDownloadBandwidthLimiter(ctx, conn, limiter), limiter)
}

func packetConnWithDownloadBandwidthWrapper(ctx context.Context, conn net.PacketConn, limiter *rate.Limiter, reverse bool) net.PacketConn {
	if reverse {
		return NewPacketConnWithUploadBandwidthLimiter(ctx, conn, limiter)
	}
	return NewPacketConnWithDownloadBandwidthLimiter(ctx, conn, limiter)
}

func packetConnWithUploadBandwidthWrapper(ctx context.Context, conn net.PacketConn, limiter *rate.Limiter, reverse bool) net.PacketConn {
	if reverse {
		return NewPacketConnWithDownloadBandwidthLimiter(ctx, conn, limiter)
	}
	return NewPacketConnWithUploadBandwidthLimiter(ctx, conn, limiter)
}

func packetConnWithDuplexBandwidthWrapper(ctx context.Context, conn net.PacketConn, limiter *rate.Limiter, reverse bool) net.PacketConn {
	return NewPacketConnWithUploadBandwidthLimiter(ctx, NewPacketConnWithDownloadBandwidthLimiter(ctx, conn, limiter), limiter)
}
