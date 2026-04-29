package bandwidth

import (
	"context"
	"net"

	"golang.org/x/time/rate"
)

type connWithDownloadBandwidthLimiter struct {
	net.Conn
	ctx     context.Context
	limiter Limiter
}

func NewConnWithDownloadBandwidthLimiter(ctx context.Context, conn net.Conn, limiter *rate.Limiter) *connWithDownloadBandwidthLimiter {
	return &connWithDownloadBandwidthLimiter{conn, ctx, limiter}
}

func (conn *connWithDownloadBandwidthLimiter) Write(p []byte) (n int, err error) {
	err = conn.limiter.WaitN(conn.ctx, len(p))
	if err != nil {
		return
	}
	return conn.Conn.Write(p)
}

type connWithUploadBandwidthLimiter struct {
	net.Conn
	ctx     context.Context
	limiter *rate.Limiter
	burst   int
}

func NewConnWithUploadBandwidthLimiter(ctx context.Context, conn net.Conn, limiter *rate.Limiter) *connWithUploadBandwidthLimiter {
	return &connWithUploadBandwidthLimiter{conn, ctx, limiter, limiter.Burst()}
}

func (conn *connWithUploadBandwidthLimiter) Read(p []byte) (n int, err error) {
	n, err = conn.Conn.Read(p)
	if err != nil {
		return
	}
	err = conn.limiter.WaitN(conn.ctx, n)
	if err != nil {
		return
	}
	return n, err
}

type connWithCloseHandler struct {
	net.Conn
	onClose CloseHandlerFunc
}

func NewConnWithCloseHandler(conn net.Conn, onClose CloseHandlerFunc) *connWithCloseHandler {
	return &connWithCloseHandler{conn, onClose}
}

func (conn *connWithCloseHandler) Close() error {
	conn.onClose()
	return conn.Conn.Close()
}

type packetConnWithDownloadBandwidthLimiter struct {
	net.PacketConn
	ctx     context.Context
	limiter *rate.Limiter
	burst   int
}

func NewPacketConnWithDownloadBandwidthLimiter(ctx context.Context, conn net.PacketConn, limiter *rate.Limiter) *packetConnWithDownloadBandwidthLimiter {
	return &packetConnWithDownloadBandwidthLimiter{conn, ctx, limiter, limiter.Burst()}
}

func (conn *packetConnWithDownloadBandwidthLimiter) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	err = conn.limiter.WaitN(conn.ctx, len(p))
	if err != nil {
		return
	}
	return conn.PacketConn.WriteTo(p, addr)
}

type packetConnWithUploadBandwidthLimiter struct {
	net.PacketConn
	ctx     context.Context
	limiter Limiter
	burst   int
}

func NewPacketConnWithUploadBandwidthLimiter(ctx context.Context, conn net.PacketConn, limiter *rate.Limiter) *packetConnWithUploadBandwidthLimiter {
	return &packetConnWithUploadBandwidthLimiter{conn, ctx, limiter, limiter.Burst()}
}

func (conn *packetConnWithUploadBandwidthLimiter) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	n, addr, err = conn.PacketConn.ReadFrom(p)
	if err != nil {
		return
	}
	err = conn.limiter.WaitN(conn.ctx, n)
	if err != nil {
		return
	}
	return
}

type packetConnWithCloseHandler struct {
	net.PacketConn
	onClose CloseHandlerFunc
}

func NewPacketConnWithCloseHandler(conn net.PacketConn, onClose CloseHandlerFunc) *packetConnWithCloseHandler {
	return &packetConnWithCloseHandler{conn, onClose}
}

func (conn *packetConnWithCloseHandler) Close() error {
	conn.onClose()
	return conn.PacketConn.Close()
}
