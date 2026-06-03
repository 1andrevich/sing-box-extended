package trusttunnel

import (
	"context"
	"errors"
	"net"
	"net/http"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/inbound"
	"github.com/sagernet/sing-box/common/listener"
	"github.com/sagernet/sing-box/common/tls"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-box/transport/trusttunnel"
	"github.com/sagernet/sing/common"
	"github.com/sagernet/sing/common/auth"
	"github.com/sagernet/sing/common/logger"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	aTLS "github.com/sagernet/sing/common/tls"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func RegisterInbound(registry *inbound.Registry) {
	inbound.Register[option.TrustTunnelInboundOptions](registry, C.TypeTrustTunnel, NewInbound)
}

type Inbound struct {
	inbound.Adapter
	ctx         context.Context
	router      adapter.Router
	logger      logger.ContextLogger
	listener    *listener.Listener
	tlsConfig   tls.ServerConfig
	service     *trusttunnel.Service
	httpServer  *http.Server
	quicService *trusttunnel.QUICService
	network     []string
}

func NewInbound(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.TrustTunnelInboundOptions) (adapter.Inbound, error) {
	if options.TLS == nil || !options.TLS.Enabled {
		return nil, C.ErrTLSRequired
	}
	tlsConfig, err := tls.NewServer(ctx, logger, common.PtrValueOrDefault(options.TLS))
	if err != nil {
		return nil, err
	}
	networkList := options.Network.Build()
	if len(networkList) == 0 {
		networkList = []string{N.NetworkTCP}
	}
	inbound := &Inbound{
		Adapter:   inbound.NewAdapter(C.TypeTrustTunnel, tag),
		ctx:       ctx,
		router:    router,
		logger:    logger,
		tlsConfig: tlsConfig,
		network:   networkList,
		listener: listener.New(listener.Options{
			Context: ctx,
			Logger:  logger,
			Listen:  options.ListenOptions,
		}),
	}
	service := trusttunnel.NewService(trusttunnel.ServiceOptions{
		Ctx:     ctx,
		Logger:  logger,
		Handler: (*inboundHandler)(inbound),
	})
	userMap := make(map[string]string, len(options.Users))
	for _, u := range options.Users {
		userMap[u.Name] = u.Password
	}
	service.UpdateUsers(userMap)
	inbound.service = service
	if common.Contains(networkList, N.NetworkUDP) {
		inbound.quicService = trusttunnel.NewQUICService(service, options.CongestionController, options.CWND, options.BBRProfile)
	}
	return inbound, nil
}

func (h *Inbound) Start(stage adapter.StartStage) error {
	if stage != adapter.StartStateStart {
		return nil
	}
	if h.tlsConfig != nil {
		err := h.tlsConfig.Start()
		if err != nil {
			return err
		}
	}
	if common.Contains(h.network, N.NetworkTCP) {
		tcpListener, err := h.listener.ListenTCP()
		if err != nil {
			return err
		}
		h.httpServer = &http.Server{
			Handler: h2c.NewHandler(h.service, &http2.Server{}),
			BaseContext: func(net.Listener) context.Context {
				return h.ctx
			},
		}
		go func() {
			var l net.Listener = tcpListener
			if h.tlsConfig != nil {
				if len(h.tlsConfig.NextProtos()) == 0 {
					h.tlsConfig.SetNextProtos([]string{http2.NextProtoTLS})
				} else if !common.Contains(h.tlsConfig.NextProtos(), http2.NextProtoTLS) {
					h.tlsConfig.SetNextProtos(append([]string{http2.NextProtoTLS}, h.tlsConfig.NextProtos()...))
				}
				l = aTLS.NewListener(tcpListener, h.tlsConfig)
			}
			sErr := h.httpServer.Serve(l)
			if sErr != nil && !errors.Is(sErr, http.ErrServerClosed) {
				h.logger.Error("HTTP server error: ", sErr)
			}
		}()
	}
	if common.Contains(h.network, N.NetworkUDP) {
		udpConn, err := h.listener.ListenUDP()
		if err != nil {
			return err
		}
		err = h.quicService.Start(h.ctx, udpConn, h.tlsConfig)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *Inbound) Close() error {
	return common.Close(
		h.listener,
		common.PtrOrNil(h.httpServer),
		h.quicService,
		h.tlsConfig,
	)
}

func (h *Inbound) UpdateUsers(users []option.TrustTunnelUser) {
	userMap := make(map[string]string, len(users))
	for _, u := range users {
		userMap[u.Name] = u.Password
	}
	h.service.UpdateUsers(userMap)
}

type inboundHandler Inbound

func (h *inboundHandler) NewConnection(ctx context.Context, conn net.Conn, metadata M.Metadata) error {
	var inboundCtx adapter.InboundContext
	inboundCtx.Inbound = h.Tag()
	inboundCtx.InboundType = h.Type()
	//nolint:staticcheck
	inboundCtx.InboundDetour = h.listener.ListenOptions().Detour
	inboundCtx.Source = metadata.Source
	inboundCtx.Destination = metadata.Destination
	if userName, _ := auth.UserFromContext[string](ctx); userName != "" {
		inboundCtx.User = userName
		h.logger.InfoContext(ctx, "[", userName, "] inbound connection to ", inboundCtx.Destination)
	} else {
		h.logger.InfoContext(ctx, "inbound connection to ", inboundCtx.Destination)
	}
	h.router.RouteConnectionEx(ctx, conn, inboundCtx, nil)
	return nil
}

func (h *inboundHandler) NewPacketConnection(ctx context.Context, conn N.PacketConn, metadata M.Metadata) error {
	var inboundCtx adapter.InboundContext
	inboundCtx.Inbound = h.Tag()
	inboundCtx.InboundType = h.Type()
	//nolint:staticcheck
	inboundCtx.InboundDetour = h.listener.ListenOptions().Detour
	inboundCtx.Source = metadata.Source
	inboundCtx.Destination = metadata.Destination
	if userName, _ := auth.UserFromContext[string](ctx); userName != "" {
		inboundCtx.User = userName
		h.logger.InfoContext(ctx, "[", userName, "] inbound packet connection to ", inboundCtx.Destination)
	} else {
		h.logger.InfoContext(ctx, "inbound packet connection to ", inboundCtx.Destination)
	}
	h.router.RoutePacketConnectionEx(ctx, conn, inboundCtx, nil)
	return nil
}
