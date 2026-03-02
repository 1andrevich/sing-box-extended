package bond

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/inbound"
	"github.com/sagernet/sing-box/common/kmutex"
	"github.com/sagernet/sing-box/common/uot"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/logger"
	N "github.com/sagernet/sing/common/network"
	"github.com/sagernet/sing/service"
)

func RegisterInbound(registry *inbound.Registry) {
	inbound.Register[option.BondInboundOptions](registry, C.TypeBond, NewInbound)
}

type Inbound struct {
	inbound.Adapter
	logger   logger.ContextLogger
	router   adapter.ConnectionRouterEx
	inbounds []adapter.Inbound
	conns    *cache.Cache

	mtx *kmutex.Kmutex[string]
}

func NewInbound(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.BondInboundOptions) (adapter.Inbound, error) {
	if len(options.Inbounds) == 0 {
		return nil, E.New("missing tags")
	}
	inbound := &Inbound{
		Adapter: inbound.NewAdapter(C.TypeBond, tag),
		logger:  logger,
		router:  uot.NewRouter(router, logger),
		conns:   cache.New(C.TCPConnectTimeout, time.Second),
		mtx:     kmutex.New[string](),
	}
	inboundRegistry := service.FromContext[adapter.InboundRegistry](ctx)
	inbounds := make([]adapter.Inbound, len(options.Inbounds))
	for i, inboundOptions := range options.Inbounds {
		inbound, err := inboundRegistry.UnsafeCreate(ctx, NewRouter(router, logger, inbound.connHandler), logger, inboundOptions.Tag, inboundOptions.Type, inboundOptions.Options)
		if err != nil {
			return nil, err
		}
		inbounds[i] = inbound
	}
	inbound.inbounds = inbounds
	inbound.conns.OnEvicted(func(s string, i interface{}) {
		inbound.mtx.Lock(s)
		defer inbound.mtx.Unlock(s)
		ratioConns := i.(map[uint8]*ratioConn)
		for _, ratioConn := range ratioConns {
			if ratioConn != nil {
				ratioConn.conn.Close()
			}
		}
	})
	return inbound, nil
}

func (h *Inbound) Start(stage adapter.StartStage) error {
	for _, inbound := range h.inbounds {
		err := inbound.Start(stage)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *Inbound) Close() error {
	errs := make([]error, 0)
	for _, inbound := range h.inbounds {
		err := inbound.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) != 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (h *Inbound) connHandler(ctx context.Context, conn net.Conn, metadata adapter.InboundContext, onClose N.CloseHandlerFunc) error {
	if metadata.Destination != Destination {
		h.router.RouteConnectionEx(ctx, conn, metadata, onClose)
		return nil
	}
	request, err := ReadRequest(conn)
	if err != nil {
		return err
	}
	requestUUID := request.UUID.String()
	h.mtx.Lock(requestUUID)
	var ratioConns map[uint8]*ratioConn
	rawRatioConns, ok := h.conns.Get(requestUUID)
	if ok {
		ratioConns = rawRatioConns.(map[uint8]*ratioConn)
	} else {
		ratioConns = make(map[uint8]*ratioConn, request.Count)
		h.conns.SetDefault(requestUUID, ratioConns)
	}
	ratioConns[request.Index] = &ratioConn{
		conn:          conn,
		downloadRatio: request.DownloadRatio,
		uploadRatio:   request.UploadRatio,
	}
	if len(ratioConns) == int(request.Count) {
		conns := make([]net.Conn, len(ratioConns))
		downloadRatios := make([]uint8, len(ratioConns))
		uploadRatios := make([]uint8, len(ratioConns))
		var totalDownloadRatio, totalUploadRatio uint8
		for index, ratioConn := range ratioConns {
			conns[index] = ratioConn.conn
			downloadRatios[index] = ratioConn.downloadRatio
			uploadRatios[index] = ratioConn.uploadRatio
			totalDownloadRatio += ratioConn.downloadRatio
			totalUploadRatio += ratioConn.uploadRatio
			delete(ratioConns, index)
		}
		if totalDownloadRatio != 100 || totalUploadRatio != 100 {
			for _, conn := range conns {
				conn.Close()
			}
			h.mtx.Unlock(requestUUID)
			return E.New("invalid ratios")
		}
		conn = NewBondedConn(conns, downloadRatios, uploadRatios)
		metadata.Inbound = h.Tag()
		metadata.InboundType = C.TypeBond
		metadata.Destination = request.Destination
		h.mtx.Unlock(requestUUID)
		h.router.RouteConnectionEx(ctx, conn, metadata, onClose)
		return nil
	}
	h.mtx.Unlock(requestUUID)
	return nil
}

type ratioConn struct {
	conn          net.Conn
	downloadRatio uint8
	uploadRatio   uint8
}
