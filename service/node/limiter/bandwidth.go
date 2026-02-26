package limiter

import (
	"sync"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/protocol/limiter/bandwidth"
	CM "github.com/sagernet/sing-box/service/manager/constant"
	"github.com/sagernet/sing-box/service/node/constant"
	E "github.com/sagernet/sing/common/exceptions"
)

type ManagedBandwidthStrategy interface {
	UpdateStrategies(strategies map[string]bandwidth.BandwidthStrategy)
}

type BandwidthLimiterManager struct {
	managers map[string]*BandwidthLimiterStrategyManager

	mtx sync.Mutex
}

func NewBandwidthLimiterManager() *BandwidthLimiterManager {
	return &BandwidthLimiterManager{
		managers: make(map[string]*BandwidthLimiterStrategyManager),
	}
}

func (m *BandwidthLimiterManager) AddBandwidthLimiterStrategyManager(outbound adapter.Outbound) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	limiter, ok := outbound.(*bandwidth.Outbound)
	if !ok {
		return E.New("invalid bandwidth limiter: ", outbound.Tag())
	}
	strategy, ok := limiter.GetStrategy().(ManagedBandwidthStrategy)
	if !ok {
		return E.New("strategy for outbound ", outbound.Tag(), " is not manager")
	}
	m.managers[outbound.Tag()] = &BandwidthLimiterStrategyManager{
		strategy:      strategy,
		strategiesMap: make(map[string]bandwidth.BandwidthStrategy),
	}
	return nil
}

func (m *BandwidthLimiterManager) GetBandwidthLimiterStrategyManager(tag string) (constant.BandwidthLimiterStrategyManager, bool) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	manager, ok := m.managers[tag]
	return manager, ok
}

func (m *BandwidthLimiterManager) GetBandwidthLimiterStrategyManagerTags() []string {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	tags := make([]string, 0, len(m.managers))
	for tag, _ := range m.managers {
		tags = append(tags, tag)
	}
	return tags
}

type BandwidthLimiterStrategyManager struct {
	strategy      ManagedBandwidthStrategy
	strategiesMap map[string]bandwidth.BandwidthStrategy

	mtx sync.Mutex
}

func (i *BandwidthLimiterStrategyManager) postUpdate() {
	i.strategy.UpdateStrategies(i.strategiesMap)
}

func (i *BandwidthLimiterStrategyManager) UpdateBandwidthLimiter(limiter CM.BandwidthLimiter) {
	i.mtx.Lock()
	defer i.mtx.Unlock()
	strategy, err := bandwidth.CreateStrategy(limiter.Strategy, limiter.Mode, limiter.ConnectionType, limiter.RawSpeed)
	if err != nil {
		return
	}
	i.strategiesMap[limiter.Username] = strategy
	i.postUpdate()
}

func (i *BandwidthLimiterStrategyManager) UpdateBandwidthLimiters(limiters []CM.BandwidthLimiter) {
	i.mtx.Lock()
	defer i.mtx.Unlock()
	clear(i.strategiesMap)
	newStrategiesMap := make(map[string]bandwidth.BandwidthStrategy)
	for _, limiter := range limiters {
		strategy, err := bandwidth.CreateStrategy(limiter.Strategy, limiter.Mode, limiter.ConnectionType, limiter.RawSpeed)
		if err != nil {
			return
		}
		newStrategiesMap[limiter.Username] = strategy
	}
	i.strategiesMap = newStrategiesMap
	i.postUpdate()
}

func (i *BandwidthLimiterStrategyManager) DeleteBandwidthLimiter(username string) {
	i.mtx.Lock()
	defer i.mtx.Unlock()
	delete(i.strategiesMap, username)
	i.postUpdate()
}
