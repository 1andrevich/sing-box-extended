package bandwidth

import (
	"context"
	"net"

	"golang.org/x/time/rate"
)

type connWithDownloadBandwidthLimiter struct {
	net.Conn
	ctx     context.Context
	limiter *rate.Limiter
	burst   int
}

func NewConnWithDownloadBandwidthLimiter(ctx context.Context, conn net.Conn, limiter *rate.Limiter) *connWithDownloadBandwidthLimiter {
	return &connWithDownloadBandwidthLimiter{conn, ctx, limiter, limiter.Burst()}
}

func (conn *connWithDownloadBandwidthLimiter) Write(p []byte) (n int, err error) {
	var nn int
	for {
		end := len(p)
		if end == 0 {
			break
		}
		if conn.burst < len(p) {
			end = conn.burst
		}
		err = conn.limiter.WaitN(conn.ctx, end)
		if err != nil {
			return
		}
		nn, err = conn.Conn.Write(p[:end])
		n += nn
		if err != nil {
			return
		}
		p = p[end:]
	}
	return
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
	if conn.burst < len(p) {
		p = p[:conn.burst]
	}
	n, err = conn.Conn.Read(p)
	if err != nil {
		return
	}
	err = conn.limiter.WaitN(conn.ctx, n)
	if err != nil {
		return
	}
	return
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
	var nn int
	for {
		end := len(p)
		if end == 0 {
			break
		}
		if conn.burst < len(p) {
			end = conn.burst
		}
		err = conn.limiter.WaitN(conn.ctx, end)
		if err != nil {
			return
		}
		nn, err = conn.PacketConn.WriteTo(p[:end], addr)
		n += nn
		if err != nil {
			return
		}
		p = p[end:]
	}
	return
}

type packetConnWithUploadBandwidthLimiter struct {
	net.PacketConn
	ctx     context.Context
	limiter *rate.Limiter
	burst   int
}

func NewPacketConnWithUploadBandwidthLimiter(ctx context.Context, conn net.PacketConn, limiter *rate.Limiter) *packetConnWithUploadBandwidthLimiter {
	return &packetConnWithUploadBandwidthLimiter{conn, ctx, limiter, limiter.Burst()}
}

func (conn *packetConnWithUploadBandwidthLimiter) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	if conn.burst < len(p) {
		p = p[:conn.burst]
	}
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
