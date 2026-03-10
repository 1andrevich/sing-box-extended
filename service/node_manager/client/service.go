package client

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/sagernet/sing-box/adapter"
	boxService "github.com/sagernet/sing-box/adapter/service"
	"github.com/sagernet/sing-box/common/dialer"
	"github.com/sagernet/sing-box/common/tls"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	CM "github.com/sagernet/sing-box/service/manager/constant"
	pb "github.com/sagernet/sing-box/service/node_manager/manager"
	"github.com/sagernet/sing/common"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

func RegisterService(registry *boxService.Registry) {
	boxService.Register[option.NodeManagerClientServiceOptions](registry, C.TypeNodeManagerClient, NewService)
}

type Service struct {
	boxService.Adapter

	ctx     context.Context
	logger  log.ContextLogger
	dialer  N.Dialer
	creds   credentials.TransportCredentials
	options option.NodeManagerClientServiceOptions

	conn *grpc.ClientConn

	mtx sync.Mutex
}

func NewService(ctx context.Context, logger log.ContextLogger, tag string, options option.NodeManagerClientServiceOptions) (adapter.Service, error) {
	outboundDialer, err := dialer.New(ctx, options.DialerOptions, options.ServerIsDomain())
	if err != nil {
		return nil, err
	}
	creds := insecure.NewCredentials()
	if options.TLS != nil {
		tlsConfig, err := tls.NewClient(ctx, logger, options.Server, common.PtrValueOrDefault(options.TLS))
		if err != nil {
			return nil, err
		}
		creds = &tlsCreds{tlsConfig}
	}
	return &Service{
		Adapter: boxService.NewAdapter(C.TypeManager, tag),
		ctx:     ctx,
		logger:  logger,
		dialer:  outboundDialer,
		creds:   creds,
		options: options,
	}, nil
}

func (s *Service) AddNode(uuid string, node CM.ConnectedNode) error {
	go func() {
		isRetry := false
		for {
			if !isRetry {
				select {
				case <-s.ctx.Done():
					return
				default:
					isRetry = true
				}
			} else {
				select {
				case <-time.After(5 * time.Second):
					break
				case <-s.ctx.Done():
					return
				}
			}
			conn, err := s.getConn()
			if err != nil {
				s.logger.Error(err)
				continue
			}
			client := pb.NewManagerClient(conn)
			stream, err := client.AddNode(s.ctx, &pb.Node{Uuid: uuid})
			if err != nil {
				s.logger.Error(err)
				continue
			}
			err = s.handler(node, stream)
			if err != nil {
				s.logger.Error(err)
				continue
			}
		}
	}()
	return nil
}

func (s *Service) AcquireLock(limiterId int, id string) (string, error) {
	conn, err := s.getConn()
	if err != nil {
		return "", err
	}
	client := pb.NewManagerClient(conn)
	lockReply, err := client.AcquireLock(s.ctx, &pb.AcquireLockRequest{LimiterId: int32(limiterId), Id: id})
	if err != nil {
		return "", err
	}
	return lockReply.HandleId, err
}

func (s *Service) RefreshLock(limiterId int, id string, handleId string) error {
	conn, err := s.getConn()
	if err != nil {
		return err
	}
	client := pb.NewManagerClient(conn)
	_, err = client.RefreshLock(s.ctx, &pb.LockData{LimiterId: int32(limiterId), Id: id, HandleId: handleId})
	return err
}

func (s *Service) ReleaseLock(limiterId int, id string, handleId string) error {
	conn, err := s.getConn()
	if err != nil {
		return err
	}
	client := pb.NewManagerClient(conn)
	_, err = client.ReleaseLock(s.ctx, &pb.LockData{LimiterId: int32(limiterId), Id: id, HandleId: handleId})
	return err
}

func (s *Service) Start(stage adapter.StartStage) error {
	return nil
}

func (s *Service) Close() error {
	return nil
}

func (s *Service) getConn() (*grpc.ClientConn, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if s.conn != nil {
		state := s.conn.GetState()
		if state != connectivity.Shutdown && state != connectivity.TransientFailure {
			return s.conn, nil
		}
	}
	for {
		conn, err := s.createConn()
		if err != nil {
			return nil, err
		}
		s.conn = conn
		return conn, nil
	}
}

func (s *Service) createConn() (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(
		s.options.ServerOptions.Build().String(),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			return s.dialer.DialContext(ctx, N.NetworkTCP, M.ParseSocksaddr(addr))
		}),
		grpc.WithTransportCredentials(s.creds),
	)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (s *Service) handler(node CM.ConnectedNode, stream grpc.ServerStreamingClient[pb.NodeData]) error {
	for {
		data, err := stream.Recv()
		if err != nil {
			return err
		}
		switch data.Op {
		case pb.OpType_updateUser:
			s.logger.DebugContext(s.ctx, "update user")
			node.UpdateUser(s.convertUser(data.Data.(*pb.NodeData_User).User))
		case pb.OpType_updateUsers:
			s.logger.DebugContext(s.ctx, "update users")
			users := data.Data.(*pb.NodeData_Users).Users.Values
			convertedUsers := make([]CM.User, len(users))
			for i, user := range users {
				convertedUsers[i] = s.convertUser(user)
			}
			node.UpdateUsers(convertedUsers)
		case pb.OpType_deleteUser:
			s.logger.DebugContext(s.ctx, "delete user")
			node.DeleteUser(s.convertUser(data.Data.(*pb.NodeData_User).User))

		case pb.OpType_updateConnectionLimiter:
			s.logger.DebugContext(s.ctx, "update connection limiter")
			node.UpdateConnectionLimiter(s.convertConnectionLimiter(data.Data.(*pb.NodeData_ConnectionLimiter).ConnectionLimiter))
		case pb.OpType_updateConnectionLimiters:
			s.logger.DebugContext(s.ctx, "update connection limiters")
			limiters := data.Data.(*pb.NodeData_ConnectionLimiters).ConnectionLimiters.Values
			convertedLimiters := make([]CM.ConnectionLimiter, len(limiters))
			for i, limiter := range limiters {
				convertedLimiters[i] = s.convertConnectionLimiter(limiter)
			}
			node.UpdateConnectionLimiters(convertedLimiters)
		case pb.OpType_deleteConnectionLimiter:
			s.logger.DebugContext(s.ctx, "delete connection limiter")
			node.DeleteConnectionLimiter(s.convertConnectionLimiter(data.Data.(*pb.NodeData_ConnectionLimiter).ConnectionLimiter))

		case pb.OpType_updateBandwidthLimiter:
			s.logger.DebugContext(s.ctx, "update bandwidth limiter")
			node.UpdateBandwidthLimiter(s.convertBandwidthLimiter(data.Data.(*pb.NodeData_BandwidthLimiter).BandwidthLimiter))
		case pb.OpType_updateBandwidthLimiters:
			s.logger.DebugContext(s.ctx, "update bandwidth limiters")
			limiters := data.Data.(*pb.NodeData_BandwidthLimiters).BandwidthLimiters.Values
			convertedLimiters := make([]CM.BandwidthLimiter, len(limiters))
			for i, limiter := range limiters {
				convertedLimiters[i] = s.convertBandwidthLimiter(limiter)
			}
			node.UpdateBandwidthLimiters(convertedLimiters)
		case pb.OpType_deleteBandwidthLimiter:
			s.logger.DebugContext(s.ctx, "delete bandwidth limiter")
			node.DeleteBandwidthLimiter(s.convertBandwidthLimiter(data.Data.(*pb.NodeData_BandwidthLimiter).BandwidthLimiter))
		}
	}
}

func (s *Service) convertUser(user *pb.User) CM.User {
	return CM.User{
		ID:       int(user.Id),
		Username: user.Username,
		Type:     user.Type,
		Inbound:  user.Inbound,
		UUID:     user.Uuid,
		Password: user.Password,
		Flow:     user.Flow,
		AlterID:  int(user.AlterId),
	}
}

func (s *Service) convertBandwidthLimiter(limiter *pb.BandwidthLimiter) CM.BandwidthLimiter {
	return CM.BandwidthLimiter{
		ID:             int(limiter.Id),
		Username:       limiter.Username,
		Outbound:       limiter.Outbound,
		Strategy:       limiter.Strategy,
		Mode:           limiter.Mode,
		ConnectionType: limiter.ConnectionType,
		Speed:          limiter.Speed,
		RawSpeed:       limiter.RawSpeed,
	}
}

func (s *Service) convertConnectionLimiter(limiter *pb.ConnectionLimiter) CM.ConnectionLimiter {
	return CM.ConnectionLimiter{
		ID:             int(limiter.Id),
		Username:       limiter.Username,
		Outbound:       limiter.Outbound,
		Strategy:       limiter.Strategy,
		ConnectionType: limiter.ConnectionType,
		LockType:       limiter.LockType,
		Count:          limiter.Count,
	}
}
