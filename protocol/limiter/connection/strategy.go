package connection

import (
	"context"
	"strconv"
	"sync"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/log"
	E "github.com/sagernet/sing/common/exceptions"
)

type (
	CloseHandlerFunc = func()

	ConnIDGetter = func(context.Context, *adapter.InboundContext) (string, bool)
	LockIDGetter = func(string) (CloseHandlerFunc, context.Context, error)

	ConnectionStrategy interface {
		request(ctx context.Context, metadata *adapter.InboundContext) (onClose CloseHandlerFunc, lockCtx context.Context, err error)
	}
)

type DefaultConnectionStrategy struct {
	connIDGetter ConnIDGetter
	lockIDGetter LockIDGetter

	mtx sync.Mutex
}

func NewDefaultConnectionStrategy(connIDGetter ConnIDGetter, lockIDGetter LockIDGetter) *DefaultConnectionStrategy {
	outbound := &DefaultConnectionStrategy{
		connIDGetter: connIDGetter,
		lockIDGetter: lockIDGetter,
	}
	return outbound
}

func (s *DefaultConnectionStrategy) request(ctx context.Context, metadata *adapter.InboundContext) (CloseHandlerFunc, context.Context, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	id, ok := s.connIDGetter(ctx, metadata)
	if !ok {
		return nil, nil, E.New("id not found")
	}
	return s.lockIDGetter(id)
}

type UsersConnectionStrategy struct {
	strategies map[string]ConnectionStrategy
	mtx        sync.Mutex
}

func NewUsersConnectionStrategy(strategies map[string]ConnectionStrategy) *UsersConnectionStrategy {
	return &UsersConnectionStrategy{
		strategies: strategies,
	}
}

func (s *UsersConnectionStrategy) request(ctx context.Context, metadata *adapter.InboundContext) (CloseHandlerFunc, context.Context, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	var user string
	if metadata != nil {
		user = metadata.User
	}
	strategy, ok := s.strategies[user]
	if ok {
		return strategy.request(ctx, metadata)
	}
	return nil, nil, E.New("user strategy not found: ", user)
}

type ManagerConnectionStrategy struct {
	*UsersConnectionStrategy
}

func NewManagerConnectionStrategy() *ManagerConnectionStrategy {
	return &ManagerConnectionStrategy{
		UsersConnectionStrategy: NewUsersConnectionStrategy(map[string]ConnectionStrategy{}),
	}
}

func (s *ManagerConnectionStrategy) UpdateStrategies(strategies map[string]ConnectionStrategy) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.strategies = strategies
}

func CreateStrategy(strategy string, connectionType string, lockIDGetter LockIDGetter) (ConnectionStrategy, error) {
	switch strategy {
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
		return NewDefaultConnectionStrategy(connIDGetter, lockIDGetter), nil
	default:
		return nil, E.New("strategy not found: ", strategy)
	}
}
