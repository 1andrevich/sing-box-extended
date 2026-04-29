package masque

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	connectip "github.com/Diniboy1123/connect-ip-go"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/logger"
	M "github.com/sagernet/sing/common/metadata"
)

type TunnelDevice interface {
	ReadPacket(buf []byte) (int, error)
	WritePacket(pkt []byte) error
}

type Tunnel struct {
	ctx          context.Context
	logger       logger.ContextLogger
	options      TunnelOptions
	tunDevice    Device
	tunnelDevice TunnelDevice
}

func NewTunnel(ctx context.Context, logger logger.ContextLogger, options TunnelOptions) (*Tunnel, error) {
	deviceOptions := DeviceOptions{
		Context:    ctx,
		Logger:     logger,
		Handler:    options.Handler,
		UDPTimeout: options.UDPTimeout,
		MTU:        1280,
		Address:    options.Address,
	}
	tunDevice, err := NewDevice(deviceOptions)
	if err != nil {
		return nil, E.Cause(err, "create MASQUE device")
	}
	return &Tunnel{
		ctx:          ctx,
		logger:       logger,
		options:      options,
		tunDevice:    tunDevice,
		tunnelDevice: NewNetstackAdapter(tunDevice),
	}, nil
}

func (e *Tunnel) Start(resolve bool) error {
	if resolve {
		err := e.tunDevice.Start()
		if err != nil {
			return err
		}
		go e.MaintainTunnel()
	}
	return nil
}

func (e *Tunnel) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	if !destination.Addr.IsValid() {
		return nil, E.Cause(os.ErrInvalid, "invalid non-IP destination")
	}
	return e.tunDevice.DialContext(ctx, network, destination)
}

func (e *Tunnel) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	if !destination.Addr.IsValid() {
		return nil, E.Cause(os.ErrInvalid, "invalid non-IP destination")
	}
	return e.tunDevice.ListenPacket(ctx, destination)
}

func (e *Tunnel) Close() error {
	return e.tunDevice.Close()
}

func (e *Tunnel) MaintainTunnel() {
	packetBufferPool := NewNetBuffer(1280)
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-e.ctx.Done():
			return
		default:
		}
		e.logger.InfoContext(e.ctx, fmt.Errorf("Establishing MASQUE connection to %s", e.options.Endpoint))
		udpConn, tr, ipConn, rsp, err := ConnectTunnel(
			e.ctx,
			e.options.Dialer,
			e.options.TLSConfig,
			DefaultQuicConfig(e.options.UDPKeepalivePeriod, e.options.UDPInitialPacketSize),
			"https://cloudflareaccess.com",
			e.options.Endpoint,
			e.options.UseHTTP2,
		)
		if err != nil {
			e.logger.InfoContext(e.ctx, fmt.Errorf("Failed to connect tunnel: %v", err))
			timer.Reset(e.options.ReconnectDelay)
			select {
			case <-e.ctx.Done():
				return
			case <-timer.C:
			}
			continue
		}
		if rsp.StatusCode != 200 {
			e.logger.InfoContext(e.ctx, fmt.Errorf("Tunnel connection failed: %s", rsp.Status))
			ipConn.Close()
			if udpConn != nil {
				udpConn.Close()
			}
			if tr != nil {
				tr.Close()
			}
			timer.Reset(e.options.ReconnectDelay)
			select {
			case <-e.ctx.Done():
				return
			case <-timer.C:
			}
			continue
		}
		e.logger.InfoContext(e.ctx, "Connected to MASQUE server")
		errChan := make(chan error, 2)
		go func() {
			for {
				buf := packetBufferPool.Get()
				n, err := e.tunnelDevice.ReadPacket(buf)
				if err != nil {
					packetBufferPool.Put(buf)
					errChan <- fmt.Errorf("failed to read from TUN device: %w", err)
					return
				}
				icmp, err := ipConn.WritePacket(buf[:n])
				if err != nil {
					packetBufferPool.Put(buf)
					if errors.As(err, new(*connectip.CloseError)) {
						errChan <- fmt.Errorf("connection closed while writing to IP connection: %w", err)
						return
					}
					e.logger.InfoContext(e.ctx, fmt.Errorf("Error writing to IP connection: %v, continuing...", err))
					continue
				}
				packetBufferPool.Put(buf)
				if len(icmp) > 0 {
					if err := e.tunnelDevice.WritePacket(icmp); err != nil {
						if errors.As(err, new(*connectip.CloseError)) {
							errChan <- fmt.Errorf("connection closed while writing ICMP to TUN device: %w", err)
							return
						}
						e.logger.InfoContext(e.ctx, fmt.Errorf("Error writing ICMP to TUN device: %v, continuing...", err))
					}
				}
			}
		}()
		go func() {
			buf := packetBufferPool.Get()
			defer packetBufferPool.Put(buf)
			for {
				n, err := ipConn.ReadPacket(buf, true)
				if err != nil {
					if e.options.UseHTTP2 {
						errChan <- fmt.Errorf("connection closed while reading from IP connection: %w", err)
						return
					}
					if errors.As(err, new(*connectip.CloseError)) {
						errChan <- fmt.Errorf("connection closed while reading from IP connection: %w", err)
						return
					}
					e.logger.InfoContext(e.ctx, fmt.Errorf("Error reading from IP connection: %v, continuing...", err))
					continue
				}
				if err := e.tunnelDevice.WritePacket(buf[:n]); err != nil {
					errChan <- fmt.Errorf("failed to write to TUN device: %w", err)
					return
				}
			}
		}()
		err = <-errChan
		e.logger.InfoContext(e.ctx, fmt.Errorf("Tunnel connection lost: %v. Reconnecting...", err))
		ipConn.Close()
		if udpConn != nil {
			udpConn.Close()
		}
		if tr != nil {
			tr.Close()
		}
		timer.Reset(e.options.ReconnectDelay)
		select {
		case <-e.ctx.Done():
			return
		case <-timer.C:
		}
	}
}
