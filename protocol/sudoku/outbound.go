package sudoku

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/outbound"
	"github.com/sagernet/sing-box/common/dialer"
	"github.com/sagernet/sing-box/common/tls"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-box/transport/sudoku"
	"github.com/sagernet/sing-box/transport/sudoku/obfs/httpmask"
	"github.com/sagernet/sing/common"
	"github.com/sagernet/sing/common/bufio"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/logger"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

func RegisterOutbound(registry *outbound.Registry) {
	outbound.Register[option.SudokuOutboundOptions](registry, C.TypeSudoku, NewOutbound)
}

type Outbound struct {
	outbound.Adapter
	logger    logger.ContextLogger
	dialer    N.Dialer
	tlsConfig tls.Config
	baseConf  sudoku.ProtocolConfig

	muxMu     sync.Mutex
	muxClient *sudoku.MultiplexClient

	httpMaskMu     sync.Mutex
	httpMaskClient *httpmask.TunnelClient
	httpMaskKey    string
}

func NewOutbound(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.SudokuOutboundOptions) (adapter.Outbound, error) {
	outboundDialer, err := dialer.New(ctx, options.DialerOptions, options.ServerIsDomain())
	if err != nil {
		return nil, err
	}

	defaultConf := sudoku.DefaultConfig()
	tableType, err := sudoku.NormalizeTableType(options.TableType)
	if err != nil {
		return nil, err
	}
	paddingMin, paddingMax := sudoku.ResolvePadding(options.PaddingMin, options.PaddingMax, defaultConf.PaddingMin, defaultConf.PaddingMax)
	enablePureDownlink := sudoku.DerefBool(options.EnablePureDownlink, defaultConf.EnablePureDownlink)

	serverAddr := options.ServerOptions.Build()

	disableHTTPMask := defaultConf.DisableHTTPMask
	httpMaskMode := defaultConf.HTTPMaskMode
	var httpMaskHost string
	var pathRoot string
	httpMaskMultiplex := defaultConf.HTTPMaskMultiplex

	if hm := options.HTTPMask; hm != nil {
		disableHTTPMask = !hm.Enabled
		if hm.Mode != "" {
			httpMaskMode = hm.Mode
		}
		httpMaskHost = hm.Host
		pathRoot = strings.TrimSpace(hm.PathRoot)
		if hm.Multiplex != "" {
			httpMaskMultiplex = hm.Multiplex
		}
	}

	baseConf := sudoku.ProtocolConfig{
		ServerAddress:           serverAddr.String(),
		Key:                     options.Key,
		AEADMethod:              defaultConf.AEADMethod,
		PaddingMin:              paddingMin,
		PaddingMax:              paddingMax,
		EnablePureDownlink:      enablePureDownlink,
		HandshakeTimeoutSeconds: defaultConf.HandshakeTimeoutSeconds,
		DisableHTTPMask:         disableHTTPMask,
		HTTPMaskMode:            httpMaskMode,
		HTTPMaskHost:            httpMaskHost,
		HTTPMaskPathRoot:        pathRoot,
		HTTPMaskMultiplex:       httpMaskMultiplex,
	}
	if options.AEADMethod != "" {
		baseConf.AEADMethod = options.AEADMethod
	}

	tables, err := sudoku.NewClientTablesWithCustomPatterns(sudoku.ClientAEADSeed(options.Key), tableType, options.CustomTable, options.CustomTables)
	if err != nil {
		return nil, E.Cause(err, "build table(s)")
	}
	if len(tables) == 1 {
		baseConf.Table = tables[0]
	} else {
		baseConf.Tables = tables
	}

	out := &Outbound{
		Adapter:  outbound.NewAdapterWithDialerOptions(C.TypeSudoku, tag, []string{N.NetworkTCP, N.NetworkUDP}, options.DialerOptions),
		logger:   logger,
		dialer:   outboundDialer,
		baseConf: baseConf,
	}
	if hm := options.HTTPMask; !disableHTTPMask && hm != nil && hm.TLS != nil && hm.TLS.Enabled {
		tlsOptions := option.OutboundTLSOptions{
			Enabled:               true,
			ServerName:            options.Server,
			Fragment:              hm.TLS.Fragment,
			FragmentFallbackDelay: hm.TLS.FragmentFallbackDelay,
			RecordFragment:        hm.TLS.RecordFragment,
			KernelTx:              hm.TLS.KernelTx,
			KernelRx:              hm.TLS.KernelRx,
		}
		out.tlsConfig, err = tls.NewClientWithOptions(tls.ClientOptions{
			Context:       ctx,
			Logger:        logger,
			ServerAddress: options.Server,
			Options:       tlsOptions,
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (h *Outbound) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	switch N.NetworkName(network) {
	case N.NetworkTCP:
		h.logger.InfoContext(ctx, "outbound connection to ", destination)
	case N.NetworkUDP:
		h.logger.InfoContext(ctx, "outbound packet connection to ", destination)
	}
	ctx, metadata := adapter.ExtendContext(ctx)
	metadata.Outbound = h.Tag()
	metadata.Destination = destination

	cfg := h.baseConf
	cfg.TargetAddress = destination.String()

	muxMode := normalizeHTTPMaskMultiplex(cfg.HTTPMaskMultiplex)
	if muxMode == "on" && !cfg.DisableHTTPMask && httpTunnelModeEnabled(cfg.HTTPMaskMode) {
		stream, err := h.dialMultiplex(ctx, cfg.TargetAddress)
		if err == nil {
			return stream, nil
		}
		return nil, err
	}

	c, err := h.dialAndHandshake(ctx, &cfg)
	if err != nil {
		return nil, err
	}

	addrBuf, err := sudoku.EncodeAddress(cfg.TargetAddress)
	if err != nil {
		c.Close()
		return nil, E.Cause(err, "encode target address")
	}
	if err = sudoku.WriteKIPMessage(c, sudoku.KIPTypeOpenTCP, addrBuf); err != nil {
		c.Close()
		return nil, E.Cause(err, "send target address")
	}

	return c, nil
}

func (h *Outbound) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	h.logger.InfoContext(ctx, "outbound packet connection to ", destination)
	ctx, metadata := adapter.ExtendContext(ctx)
	metadata.Outbound = h.Tag()
	metadata.Destination = destination

	cfg := h.baseConf
	cfg.TargetAddress = destination.String()

	c, err := h.dialAndHandshake(ctx, &cfg)
	if err != nil {
		return nil, err
	}

	if err = sudoku.WriteKIPMessage(c, sudoku.KIPTypeStartUoT, nil); err != nil {
		c.Close()
		return nil, E.Cause(err, "start uot")
	}

	return bufio.NewBindPacketConn(sudoku.NewUoTPacketConn(c), destination), nil
}

func (h *Outbound) Close() error {
	h.resetMuxClient()
	h.resetHTTPMaskClient()
	return common.Close(h.tlsConfig)
}

func (h *Outbound) InterfaceUpdated() {
	h.resetMuxClient()
	h.resetHTTPMaskClient()
}

func (h *Outbound) dialAndHandshake(ctx context.Context, cfg *sudoku.ProtocolConfig) (net.Conn, error) {
	handshakeCfg := *cfg
	if !handshakeCfg.DisableHTTPMask && httpTunnelModeEnabled(handshakeCfg.HTTPMaskMode) {
		handshakeCfg.DisableHTTPMask = true
	}

	upgrade := func(raw net.Conn) (net.Conn, error) {
		return sudoku.ClientHandshake(raw, &handshakeCfg)
	}

	var c net.Conn
	var err error
	var handshakeDone bool

	if !cfg.DisableHTTPMask && httpTunnelModeEnabled(cfg.HTTPMaskMode) {
		muxMode := normalizeHTTPMaskMultiplex(cfg.HTTPMaskMultiplex)
		if muxMode == "auto" && strings.ToLower(strings.TrimSpace(cfg.HTTPMaskMode)) != "ws" {
			if client, cerr := h.getOrCreateHTTPMaskClient(cfg); cerr == nil && client != nil {
				c, err = client.DialTunnel(ctx, httpmask.TunnelDialOptions{
					Mode:         cfg.HTTPMaskMode,
					TLSConfig:    h.httpMaskTLSConfig(),
					HostOverride: cfg.HTTPMaskHost,
					PathRoot:     cfg.HTTPMaskPathRoot,
					AuthKey:      sudoku.ClientAEADSeed(cfg.Key),
					Upgrade:      upgrade,
					Multiplex:    cfg.HTTPMaskMultiplex,
					DialContext:  h.dialRaw,
				})
				if err != nil {
					h.resetHTTPMaskClient()
				}
			}
		}
		if c == nil && err == nil {
			c, err = sudoku.DialHTTPMaskTunnel(ctx, cfg.ServerAddress, cfg, h.dialRaw, upgrade)
		}
		if err == nil && c != nil {
			handshakeDone = true
		}
	}
	if c == nil && err == nil {
		c, err = h.dialer.DialContext(ctx, N.NetworkTCP, M.ParseSocksaddr(cfg.ServerAddress))
	}
	if err != nil {
		return nil, E.Cause(err, "connect to ", cfg.ServerAddress)
	}

	if !handshakeDone {
		c, err = sudoku.ClientHandshake(c, &handshakeCfg)
		if err != nil {
			common.Close(c)
			return nil, err
		}
	}

	return c, nil
}

func (h *Outbound) dialRaw(ctx context.Context, network, addr string) (net.Conn, error) {
	return h.dialer.DialContext(ctx, network, M.ParseSocksaddr(addr))
}

func (h *Outbound) httpMaskTLSConfig() httpmask.TLSClientConfig {
	if h.tlsConfig == nil {
		return nil
	}
	return tlsConfigAdapter{h.tlsConfig}
}

type tlsConfigAdapter struct {
	config tls.Config
}

func (a tlsConfigAdapter) Client(conn net.Conn) (net.Conn, error) {
	return a.config.Client(conn)
}

func (h *Outbound) dialMultiplex(ctx context.Context, targetAddress string) (net.Conn, error) {
	for attempt := 0; attempt < 2; attempt++ {
		client, err := h.getOrCreateMuxClient(ctx)
		if err != nil {
			return nil, err
		}
		stream, err := client.Dial(ctx, targetAddress)
		if err != nil {
			h.resetMuxClient()
			continue
		}
		return stream, nil
	}
	return nil, fmt.Errorf("multiplex open stream failed")
}

func (h *Outbound) getOrCreateMuxClient(ctx context.Context) (*sudoku.MultiplexClient, error) {
	h.muxMu.Lock()
	defer h.muxMu.Unlock()

	if h.muxClient != nil && !h.muxClient.IsClosed() {
		return h.muxClient, nil
	}

	baseCfg := h.baseConf
	baseConn, err := h.dialAndHandshake(ctx, &baseCfg)
	if err != nil {
		return nil, err
	}

	client, err := sudoku.StartMultiplexClient(baseConn)
	if err != nil {
		baseConn.Close()
		return nil, err
	}
	h.muxClient = client
	return client, nil
}

func (h *Outbound) resetMuxClient() {
	h.muxMu.Lock()
	defer h.muxMu.Unlock()
	if h.muxClient != nil {
		h.muxClient.Close()
		h.muxClient = nil
	}
}

func (h *Outbound) getOrCreateHTTPMaskClient(cfg *sudoku.ProtocolConfig) (*httpmask.TunnelClient, error) {
	key := cfg.ServerAddress + "|" + fmt.Sprint(h.tlsConfig != nil) + "|" + strings.TrimSpace(cfg.HTTPMaskHost)

	h.httpMaskMu.Lock()
	if h.httpMaskClient != nil && h.httpMaskKey == key {
		client := h.httpMaskClient
		h.httpMaskMu.Unlock()
		return client, nil
	}
	h.httpMaskMu.Unlock()

	client, err := httpmask.NewTunnelClient(cfg.ServerAddress, httpmask.TunnelClientOptions{
		TLSConfig:    h.httpMaskTLSConfig(),
		HostOverride: cfg.HTTPMaskHost,
		DialContext:  h.dialRaw,
		MaxIdleConns: 32,
	})
	if err != nil {
		return nil, err
	}

	h.httpMaskMu.Lock()
	defer h.httpMaskMu.Unlock()
	if h.httpMaskClient != nil && h.httpMaskKey == key {
		client.CloseIdleConnections()
		return h.httpMaskClient, nil
	}
	if h.httpMaskClient != nil {
		h.httpMaskClient.CloseIdleConnections()
	}
	h.httpMaskClient = client
	h.httpMaskKey = key
	return client, nil
}

func (h *Outbound) resetHTTPMaskClient() {
	h.httpMaskMu.Lock()
	defer h.httpMaskMu.Unlock()
	if h.httpMaskClient != nil {
		h.httpMaskClient.CloseIdleConnections()
		h.httpMaskClient = nil
		h.httpMaskKey = ""
	}
}

func normalizeHTTPMaskMultiplex(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "off":
		return "off"
	case "auto":
		return "auto"
	case "on":
		return "on"
	default:
		return "off"
	}
}

func httpTunnelModeEnabled(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "stream", "poll", "auto", "ws":
		return true
	default:
		return false
	}
}
