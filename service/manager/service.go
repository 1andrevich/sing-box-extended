//go:build with_manager

package manager

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofrs/uuid/v5"
	"github.com/patrickmn/go-cache"
	"github.com/sagernet/sing-box/adapter"
	boxService "github.com/sagernet/sing-box/adapter/service"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-box/service/manager/constant"
	"github.com/sagernet/sing-box/service/manager/repository/postgresql"
	E "github.com/sagernet/sing/common/exceptions"
)

func RegisterService(registry *boxService.Registry) {
	boxService.Register[option.ManagerServiceOptions](registry, C.TypeManager, NewService)
}

type Service struct {
	boxService.Adapter
	ctx        context.Context
	logger     log.ContextLogger
	repository constant.Repository
	nodes      map[string]constant.ConnectedNode

	limiterLocks map[int]map[string]*cache.Cache

	userValidator    *validator.Validate
	defaultValidator *validator.Validate

	mtx         sync.RWMutex
	connLockMtx sync.Mutex
}

func NewService(ctx context.Context, logger log.ContextLogger, tag string, options option.ManagerServiceOptions) (adapter.Service, error) {
	var repository constant.Repository
	var err error
	switch options.Database.Driver {
	case "postgresql":
		repository, err = postgresql.NewPostgreSQLRepository(ctx, options.Database.DSN)
		if err != nil {
			return nil, err
		}
	default:
		return nil, E.New("unknown driver \"", options.Database.Driver, "\"")
	}
	userValidator := validator.New()
	userValidator.RegisterStructValidation(func(sl validator.StructLevel) {
		user := sl.Current().Interface().(constant.UserCreate)
		switch user.Type {
		case "vless":
			if user.UUID == "" {
				sl.ReportError(user.UUID, "uuid", "UUID", "required", "")
			}
		case "vmess":
			if user.UUID == "" {
				sl.ReportError(user.UUID, "uuid", "UUID", "required", "")
			}
			if user.AlterID == 0 {
				sl.ReportError(user.AlterID, "alter_id", "AlterID", "required", "")
			}
		case "trojan", "shadowsocks", "hysteria", "hysteria2":
			if user.Password == "" {
				sl.ReportError(user.Password, "password", "Password", "required", "")
			}
		case "tuic":
			if user.UUID == "" {
				sl.ReportError(user.UUID, "uuid", "UUID", "required", "")
			}
			if user.Password == "" {
				sl.ReportError(user.Password, "password", "Password", "required", "")
			}
		}
	}, constant.UserCreate{})
	return &Service{
		Adapter:          boxService.NewAdapter(C.TypeManager, tag),
		ctx:              ctx,
		logger:           logger,
		repository:       repository,
		nodes:            make(map[string]constant.ConnectedNode, 0),
		limiterLocks:     make(map[int]map[string]*cache.Cache),
		userValidator:    userValidator,
		defaultValidator: validator.New(),
	}, nil
}

func (s *Service) CreateSquad(node constant.SquadCreate) (constant.Squad, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	err := s.defaultValidator.Struct(node)
	if err != nil {
		return constant.Squad{}, err
	}
	createdSquad, err := s.repository.CreateSquad(node)
	if err != nil {
		return createdSquad, err
	}
	return createdSquad, nil
}

func (s *Service) GetSquads(filters map[string][]string) ([]constant.Squad, error) {
	return s.repository.GetSquads(filters)
}

func (s *Service) GetSquadsCount(filters map[string][]string) (int, error) {
	return s.repository.GetSquadsCount(filters)
}

func (s *Service) GetSquad(id int) (constant.Squad, error) {
	return s.repository.GetSquad(id)
}

func (s *Service) UpdateSquad(id int, squad constant.SquadUpdate) (constant.Squad, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	err := s.defaultValidator.Struct(squad)
	if err != nil {
		return constant.Squad{}, err
	}
	updatedSquad, err := s.repository.UpdateSquad(id, squad)
	if err != nil {
		return updatedSquad, err
	}
	return updatedSquad, nil
}

func (s *Service) DeleteSquad(id int) (constant.Squad, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	deletedSquad, err := s.repository.DeleteSquad(id)
	if err != nil {
		return deletedSquad, err
	}
	return deletedSquad, nil
}

func (s *Service) CreateNode(node constant.NodeCreate) (constant.Node, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	err := s.defaultValidator.Struct(node)
	if err != nil {
		return constant.Node{}, err
	}
	createdNode, err := s.repository.CreateNode(node)
	if err != nil {
		return createdNode, err
	}
	return createdNode, nil
}

func (s *Service) GetNodes(filters map[string][]string) ([]constant.Node, error) {
	return s.repository.GetNodes(filters)
}

func (s *Service) GetNodesCount(filters map[string][]string) (int, error) {
	return s.repository.GetNodesCount(filters)
}

func (s *Service) GetNode(uuid string) (constant.Node, error) {
	return s.repository.GetNode(uuid)
}

func (s *Service) GetNodeStatus(uuid string) string {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	node, ok := s.nodes[uuid]
	if !ok || !node.IsOnline() {
		return "offline"
	}
	return "online"
}

func (s *Service) UpdateNode(uuid string, node constant.NodeUpdate) (constant.Node, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	err := s.defaultValidator.Struct(node)
	if err != nil {
		return constant.Node{}, err
	}
	updatedNode, err := s.repository.UpdateNode(uuid, node)
	if err != nil {
		return updatedNode, err
	}
	return updatedNode, nil
}

func (s *Service) DeleteNode(uuid string) (constant.Node, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	deletedNode, err := s.repository.DeleteNode(uuid)
	if err != nil {
		return deletedNode, err
	}
	node, ok := s.nodes[uuid]
	if ok {
		node.Close()
		delete(s.nodes, uuid)
	}
	return deletedNode, nil
}

func (s *Service) CreateUser(user constant.UserCreate) (constant.User, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	err := s.userValidator.Struct(user)
	if err != nil {
		return constant.User{}, err
	}
	createdUser, err := s.repository.CreateUser(user)
	if err != nil {
		return createdUser, err
	}
	nodes, err := s.repository.GetNodes(map[string][]string{
		"squad_id_in": convertIntSliceToStringSlice(createdUser.SquadIDs),
	})
	if err != nil {
		s.closeAllNodes()
		return createdUser, err
	}
	for _, node := range nodes {
		if node, ok := s.nodes[node.UUID]; ok {
			node.UpdateUser(createdUser)
		}
	}
	return createdUser, nil
}

func (s *Service) GetUsers(filters map[string][]string) ([]constant.User, error) {
	return s.repository.GetUsers(filters)
}

func (s *Service) GetUsersCount(filters map[string][]string) (int, error) {
	return s.repository.GetUsersCount(filters)
}

func (s *Service) GetUser(id int) (constant.User, error) {
	return s.repository.GetUser(id)
}

func (s *Service) UpdateUser(id int, user constant.UserUpdate) (constant.User, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	updatedUser, err := s.repository.UpdateUser(id, user)
	if err != nil {
		return updatedUser, err
	}
	nodes, err := s.repository.GetNodes(map[string][]string{
		"squad_id_in": convertIntSliceToStringSlice(updatedUser.SquadIDs),
	})
	if err != nil {
		s.closeAllNodes()
		return updatedUser, err
	}
	for _, node := range nodes {
		if node, ok := s.nodes[node.UUID]; ok {
			node.UpdateUser(updatedUser)
		}
	}
	return updatedUser, nil
}

func (s *Service) DeleteUser(id int) (constant.User, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	deletedUser, err := s.repository.DeleteUser(id)
	if err != nil {
		return deletedUser, err
	}
	nodes, err := s.repository.GetNodes(map[string][]string{
		"squad_id_in": convertIntSliceToStringSlice(deletedUser.SquadIDs),
	})
	if err != nil {
		s.closeAllNodes()
		return deletedUser, err
	}
	for _, node := range nodes {
		if node, ok := s.nodes[node.UUID]; ok {
			node.DeleteUser(deletedUser)
		}
	}
	return deletedUser, nil
}

func (s *Service) CreateConnectionLimiter(limiter constant.ConnectionLimiterCreate) (constant.ConnectionLimiter, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	err := s.defaultValidator.Struct(limiter)
	if err != nil {
		return constant.ConnectionLimiter{}, err
	}
	createdLimiter, err := s.repository.CreateConnectionLimiter(limiter)
	if err != nil {
		return createdLimiter, err
	}
	nodes, err := s.repository.GetNodes(map[string][]string{
		"squad_id_in": convertIntSliceToStringSlice(createdLimiter.SquadIDs),
	})
	if err != nil {
		s.closeAllNodes()
		return createdLimiter, err
	}
	for _, node := range nodes {
		if node, ok := s.nodes[node.UUID]; ok {
			node.UpdateConnectionLimiter(createdLimiter)
		}
	}
	return createdLimiter, nil
}

func (s *Service) GetConnectionLimiters(filters map[string][]string) ([]constant.ConnectionLimiter, error) {
	return s.repository.GetConnectionLimiters(filters)
}

func (s *Service) GetConnectionLimitersCount(filters map[string][]string) (int, error) {
	return s.repository.GetConnectionLimitersCount(filters)
}

func (s *Service) GetConnectionLimiter(id int) (constant.ConnectionLimiter, error) {
	return s.repository.GetConnectionLimiter(id)
}

func (s *Service) UpdateConnectionLimiter(id int, limiter constant.ConnectionLimiterUpdate) (constant.ConnectionLimiter, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	err := s.defaultValidator.Struct(limiter)
	if err != nil {
		return constant.ConnectionLimiter{}, err
	}
	updatedLimiter, err := s.repository.UpdateConnectionLimiter(id, limiter)
	if err != nil {
		return updatedLimiter, err
	}
	nodes, err := s.repository.GetNodes(map[string][]string{
		"squad_id_in": convertIntSliceToStringSlice(updatedLimiter.SquadIDs),
	})
	if err != nil {
		s.closeAllNodes()
		return updatedLimiter, err
	}
	for _, node := range nodes {
		if node, ok := s.nodes[node.UUID]; ok {
			node.UpdateConnectionLimiter(updatedLimiter)
		}
	}
	if limiter.LockType != "manager" {
		s.connLockMtx.Lock()
		defer s.connLockMtx.Unlock()
		delete(s.limiterLocks, id)
	}
	return updatedLimiter, nil
}

func (s *Service) DeleteConnectionLimiter(id int) (constant.ConnectionLimiter, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	deletedLimiter, err := s.repository.DeleteConnectionLimiter(id)
	if err != nil {
		return deletedLimiter, err
	}
	nodes, err := s.repository.GetNodes(map[string][]string{
		"squad_id_in": convertIntSliceToStringSlice(deletedLimiter.SquadIDs),
	})
	if err != nil {
		s.closeAllNodes()
		return deletedLimiter, err
	}
	for _, node := range nodes {
		if node, ok := s.nodes[node.UUID]; ok {
			node.DeleteConnectionLimiter(deletedLimiter)
		}
	}
	if deletedLimiter.LockType == "manager" {
		s.connLockMtx.Lock()
		defer s.connLockMtx.Unlock()
		delete(s.limiterLocks, id)
	}
	return deletedLimiter, nil
}

func (s *Service) CreateBandwidthLimiter(limiter constant.BandwidthLimiterCreate) (constant.BandwidthLimiter, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	err := s.defaultValidator.Struct(limiter)
	if err != nil {
		return constant.BandwidthLimiter{}, err
	}
	createdLimiter, err := s.repository.CreateBandwidthLimiter(limiter)
	if err != nil {
		return createdLimiter, err
	}
	nodes, err := s.repository.GetNodes(map[string][]string{
		"squad_id_in": convertIntSliceToStringSlice(createdLimiter.SquadIDs),
	})
	if err != nil {
		s.closeAllNodes()
		return createdLimiter, err
	}
	for _, node := range nodes {
		if node, ok := s.nodes[node.UUID]; ok {
			node.UpdateBandwidthLimiter(createdLimiter)
		}
	}
	return createdLimiter, nil
}

func (s *Service) GetBandwidthLimiters(filters map[string][]string) ([]constant.BandwidthLimiter, error) {
	return s.repository.GetBandwidthLimiters(filters)
}

func (s *Service) GetBandwidthLimitersCount(filters map[string][]string) (int, error) {
	return s.repository.GetBandwidthLimitersCount(filters)
}

func (s *Service) GetBandwidthLimiter(id int) (constant.BandwidthLimiter, error) {
	return s.repository.GetBandwidthLimiter(id)
}

func (s *Service) UpdateBandwidthLimiter(id int, limiter constant.BandwidthLimiterUpdate) (constant.BandwidthLimiter, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	err := s.defaultValidator.Struct(limiter)
	if err != nil {
		return constant.BandwidthLimiter{}, err
	}
	updatedLimiter, err := s.repository.UpdateBandwidthLimiter(id, limiter)
	if err != nil {
		return updatedLimiter, err
	}
	nodes, err := s.repository.GetNodes(map[string][]string{
		"squad_id_in": convertIntSliceToStringSlice(updatedLimiter.SquadIDs),
	})
	if err != nil {
		s.closeAllNodes()
		return updatedLimiter, err
	}
	for _, node := range nodes {
		if node, ok := s.nodes[node.UUID]; ok {
			node.UpdateBandwidthLimiter(updatedLimiter)
		}
	}
	return updatedLimiter, nil
}

func (s *Service) DeleteBandwidthLimiter(id int) (constant.BandwidthLimiter, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	deletedLimiter, err := s.repository.DeleteBandwidthLimiter(id)
	if err != nil {
		return deletedLimiter, err
	}
	nodes, err := s.repository.GetNodes(map[string][]string{
		"squad_id_in": convertIntSliceToStringSlice(deletedLimiter.SquadIDs),
	})
	if err != nil {
		s.closeAllNodes()
		return deletedLimiter, err
	}
	for _, node := range nodes {
		if node, ok := s.nodes[node.UUID]; ok {
			node.DeleteBandwidthLimiter(deletedLimiter)
		}
	}
	return deletedLimiter, nil
}

func (s *Service) AddNode(uuid string, node constant.ConnectedNode) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	var node_ constant.Node
	var err error
	node_, err = s.repository.GetNode(uuid)
	if err != nil {
		return err
	}
	squadIDs := convertIntSliceToStringSlice(node_.SquadIDs)
	users, err := s.repository.GetUsers(map[string][]string{
		"squad_id_in": squadIDs,
	})
	if err != nil {
		return err
	}
	node.UpdateUsers(users)
	bandwidthLimiters, err := s.repository.GetBandwidthLimiters(map[string][]string{
		"squad_id_in": squadIDs,
	})
	if err != nil {
		return err
	}
	node.UpdateBandwidthLimiters(bandwidthLimiters)
	connectionLimiters, err := s.repository.GetConnectionLimiters(map[string][]string{
		"squad_id_in": squadIDs,
	})
	if err != nil {
		return err
	}
	node.UpdateConnectionLimiters(connectionLimiters)
	s.nodes[uuid] = node
	return nil
}

func (s *Service) AcquireLock(limiterId int, id string) (string, error) {
	s.connLockMtx.Lock()
	defer s.connLockMtx.Unlock()
	limiter, err := s.repository.GetConnectionLimiter(limiterId)
	if err != nil {
		return "", err
	}
	if limiter.LockType != "manager" {
		return "", E.New("invalid lock type")
	}
	locks, ok := s.limiterLocks[limiterId]
	if !ok {
		locks = make(map[string]*cache.Cache)
		s.limiterLocks[limiter.ID] = locks
	}
	lock, ok := locks[id]
	if !ok {
		if len(locks) == int(limiter.Count) {
			return "", E.New("not enough free locks")
		}
		lock = cache.New(time.Second*30, time.Second)
		lock.OnEvicted(func(_ string, _ interface{}) {
			s.connLockMtx.Lock()
			defer s.connLockMtx.Unlock()
			if lock.ItemCount() == 0 {
				delete(locks, id)
			}
		})
		locks[id] = lock
	}
	handleID, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	lock.SetDefault(handleID.String(), new(struct{}))
	return handleID.String(), nil
}

func (s *Service) RefreshLock(limiterId int, id string, handleId string) error {
	s.connLockMtx.Lock()
	defer s.connLockMtx.Unlock()
	locks, ok := s.limiterLocks[limiterId]
	if !ok {
		return E.New("limiter not found")
	}
	lock, ok := locks[id]
	if !ok {
		return E.New("lock not found")
	}
	err := lock.Replace(handleId, new(struct{}), time.Second*30)
	return err
}

func (s *Service) ReleaseLock(limiterId int, id string, handleId string) error {
	s.connLockMtx.Lock()
	defer s.connLockMtx.Unlock()
	locks, ok := s.limiterLocks[limiterId]
	if !ok {
		return E.New("limiter not found")
	}
	lock, ok := locks[id]
	if !ok {
		return E.New("lock not found")
	}
	go lock.Delete(handleId)
	return nil
}

func (s *Service) Start(stage adapter.StartStage) error {
	return nil
}

func (s *Service) Close() error {
	return nil
}

func (s *Service) closeAllNodes() {
	for _, node := range s.nodes {
		node.Close()
	}
}

func convertIntSliceToStringSlice(values []int) []string {
	result := make([]string, len(values))
	for i, v := range values {
		result[i] = strconv.Itoa(v)
	}
	return result
}
