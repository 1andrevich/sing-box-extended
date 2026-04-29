package node

import (
	"context"
	"sync"

	"github.com/sagernet/sing-box/adapter"
	boxService "github.com/sagernet/sing-box/adapter/service"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	CM "github.com/sagernet/sing-box/service/manager/constant"
	"github.com/sagernet/sing-box/service/node/constant"
	"github.com/sagernet/sing-box/service/node/inbound"
	"github.com/sagernet/sing-box/service/node/limiter"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/service"
)

func RegisterService(registry *boxService.Registry) {
	boxService.Register[option.NodeServiceOptions](registry, C.TypeNode, NewService)
}

type Service struct {
	boxService.Adapter
	ctx               context.Context
	logger            log.ContextLogger
	inboundManagers   map[string]constant.InboundManager
	bandwidthManager  constant.BandwidthLimiterManager
	connectionManager constant.ConnectionLimiterManager
	options           option.NodeServiceOptions

	mtx sync.Mutex
}

func NewService(ctx context.Context, logger log.ContextLogger, tag string, options option.NodeServiceOptions) (adapter.Service, error) {
	return &Service{
		Adapter: boxService.NewAdapter(C.TypeManager, tag),
		ctx:     ctx,
		logger:  logger,
		options: options,
	}, nil
}

func (s *Service) Start(stage adapter.StartStage) error {
	if stage != adapter.StartStateStart {
		return nil
	}
	boxManager := service.FromContext[adapter.ServiceManager](s.ctx)
	serviceManager, ok := boxManager.Get(s.options.Manager)
	if !ok {
		return E.New("manager ", s.options.Manager, " not found")
	}
	nodeManager, ok := serviceManager.(CM.NodeManager)
	if !ok {
		return E.New("invalid ", s.options.Manager, " manager")
	}
	inboundManager := service.FromContext[adapter.InboundManager](s.ctx)
	outboundManager := service.FromContext[adapter.OutboundManager](s.ctx)
	s.inboundManagers = map[string]constant.InboundManager{
		"hysteria":  inbound.NewHysteriaManager(),
		"hysteria2": inbound.NewHysteria2Manager(),
		"trojan":    inbound.NewTrojanManager(),
		"tuic":      inbound.NewTUICManager(),
		"vless":     inbound.NewVLESSManager(),
		"vmess":     inbound.NewVMessManager(),
	}
	s.connectionManager = limiter.NewConnectionLimiterManager(nodeManager, s.logger)
	s.bandwidthManager = limiter.NewBandwidthLimiterManager()
	for _, tag := range s.options.Inbounds {
		inbound, ok := inboundManager.Get(tag)
		if !ok {
			return E.New("inbound ", tag, " not found")
		}
		inboundManager, ok := s.inboundManagers[inbound.Type()]
		if !ok {
			return E.New("inbound manager for ", tag, " not found")
		}
		err := inboundManager.AddUserManager(inbound)
		if err != nil {
			return err
		}
	}
	for _, limiter := range s.options.ConnectionLimiters {
		outbound, ok := outboundManager.Outbound(limiter)
		if !ok {
			return E.New("outbound ", limiter, " not found")
		}
		err := s.connectionManager.AddConnectionLimiterStrategyManager(outbound)
		if err != nil {
			return err
		}
	}
	for _, limiter := range s.options.BandwidthLimiters {
		outbound, ok := outboundManager.Outbound(limiter)
		if !ok {
			return E.New("outbound ", limiter, " not found")
		}
		err := s.bandwidthManager.AddBandwidthLimiterStrategyManager(outbound)
		if err != nil {
			return err
		}
	}
	return nodeManager.AddNode(s.options.UUID, s)
}

func (s *Service) UpdateUser(user CM.User) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	manager, ok := s.inboundManagers[user.Type]
	if !ok {
		return
	}
	userManager, ok := manager.GetUserManager(user.Inbound)
	if !ok {
		return
	}
	userManager.UpdateUser(user)
}

func (s *Service) UpdateUsers(users []CM.User) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	typedUsers := make(map[string][]CM.User)
	for _, user := range users {
		u, ok := typedUsers[user.Type]
		if !ok {
			typedUsers[user.Type] = make([]CM.User, 0)
		}
		typedUsers[user.Type] = append(u, user)
	}
	for type_, users := range typedUsers {
		manager, ok := s.inboundManagers[type_]
		if !ok {
			continue
		}
		for _, user := range users {
			userManager, ok := manager.GetUserManager(user.Inbound)
			if !ok {
				continue
			}
			userManager.UpdateUsers(users)
		}
	}
}

func (s *Service) DeleteUser(user CM.User) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	manager, ok := s.inboundManagers[user.Type]
	if !ok {
		return
	}
	userManager, ok := manager.GetUserManager(user.Inbound)
	if !ok {
		return
	}
	userManager.DeleteUser(user.Username)
}

func (s *Service) UpdateConnectionLimiter(limiter CM.ConnectionLimiter) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	manager, ok := s.connectionManager.GetConnectionLimiterStrategyManager(limiter.Outbound)
	if !ok {
		return
	}
	manager.UpdateConnectionLimiter(limiter)
}

func (s *Service) UpdateConnectionLimiters(limiters []CM.ConnectionLimiter) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	for _, limiter := range limiters {
		manager, ok := s.connectionManager.GetConnectionLimiterStrategyManager(limiter.Outbound)
		if !ok {
			continue
		}
		manager.UpdateConnectionLimiters(limiters)
	}
}

func (s *Service) DeleteConnectionLimiter(limiter CM.ConnectionLimiter) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	manager, ok := s.connectionManager.GetConnectionLimiterStrategyManager(limiter.Outbound)
	if !ok {
		return
	}
	manager.DeleteConnectionLimiter(limiter.Username)
}

func (s *Service) UpdateBandwidthLimiter(limiter CM.BandwidthLimiter) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	manager, ok := s.bandwidthManager.GetBandwidthLimiterStrategyManager(limiter.Outbound)
	if !ok {
		return
	}
	manager.UpdateBandwidthLimiter(limiter)
}

func (s *Service) UpdateBandwidthLimiters(limiters []CM.BandwidthLimiter) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	for _, limiter := range limiters {
		manager, ok := s.bandwidthManager.GetBandwidthLimiterStrategyManager(limiter.Outbound)
		if !ok {
			continue
		}
		manager.UpdateBandwidthLimiters(limiters)
	}
}

func (s *Service) DeleteBandwidthLimiter(limiter CM.BandwidthLimiter) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	manager, ok := s.bandwidthManager.GetBandwidthLimiterStrategyManager(limiter.Outbound)
	if !ok {
		return
	}
	manager.DeleteBandwidthLimiter(limiter.Username)
}

func (s *Service) IsLocal() bool {
	return true
}

func (s *Service) IsOnline() bool {
	return true
}

func (s *Service) Close() error {
	return nil
}
