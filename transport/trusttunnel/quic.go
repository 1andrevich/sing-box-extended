package trusttunnel

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/sagernet/quic-go"
	"github.com/sagernet/quic-go/congestion"
	"github.com/sagernet/quic-go/http3"
	"github.com/sagernet/sing-box/common/tls"
	E "github.com/sagernet/sing/common/exceptions"
	qtls "github.com/sagernet/sing-quic"
	"github.com/sagernet/sing-quic/congestion_bbr1"
	"github.com/sagernet/sing-quic/congestion_bbr2"
	congestion_meta1 "github.com/sagernet/sing-quic/congestion_meta1"
	congestion_meta2 "github.com/sagernet/sing-quic/congestion_meta2"
	"github.com/sagernet/sing/common/ntp"
)

func NewCongestionControl(name string, cwnd int, bbrProfile string, timeFunc func() time.Time) (func(conn *quic.Conn) congestion.CongestionControl, error) {
	if timeFunc == nil {
		timeFunc = time.Now
	}
	if cwnd == 0 {
		cwnd = 32
	}
	switch name {
	case "", "bbr":
		return func(conn *quic.Conn) congestion.CongestionControl {
			return congestion_meta2.NewBbrSender(
				congestion_meta2.DefaultClock{TimeFunc: timeFunc},
				congestion.ByteCount(conn.Config().InitialPacketSize),
				congestion.ByteCount(cwnd)*congestion.ByteCount(conn.Config().InitialPacketSize),
			)
		}, nil
	case "bbr_standard":
		return func(conn *quic.Conn) congestion.CongestionControl {
			return congestion_bbr1.NewBbrSender(
				congestion_bbr1.DefaultClock{TimeFunc: timeFunc},
				congestion.ByteCount(conn.Config().InitialPacketSize),
				congestion_bbr1.InitialCongestionWindowPackets,
				congestion_bbr1.MaxCongestionWindowPackets,
			)
		}, nil
	case "bbr2":
		return func(conn *quic.Conn) congestion.CongestionControl {
			return congestion_bbr2.NewBBR2Sender(
				congestion_bbr2.DefaultClock{TimeFunc: timeFunc},
				congestion.ByteCount(conn.Config().InitialPacketSize),
				0,
				false,
			)
		}, nil
	case "bbr2_variant":
		return func(conn *quic.Conn) congestion.CongestionControl {
			return congestion_bbr2.NewBBR2Sender(
				congestion_bbr2.DefaultClock{TimeFunc: timeFunc},
				congestion.ByteCount(conn.Config().InitialPacketSize),
				32*congestion.ByteCount(conn.Config().InitialPacketSize),
				true,
			)
		}, nil
	case "cubic":
		return func(conn *quic.Conn) congestion.CongestionControl {
			return congestion_meta1.NewCubicSender(
				congestion_meta1.DefaultClock{TimeFunc: timeFunc},
				congestion.ByteCount(conn.Config().InitialPacketSize),
				false,
			)
		}, nil
	case "reno":
		return func(conn *quic.Conn) congestion.CongestionControl {
			return congestion_meta1.NewCubicSender(
				congestion_meta1.DefaultClock{TimeFunc: timeFunc},
				congestion.ByteCount(conn.Config().InitialPacketSize),
				true,
			)
		}, nil
	default:
		return nil, E.New("unknown congestion control: ", name)
	}
}

type QUICService struct {
	service           *Service
	h3Server          *http3.Server
	udpConn           net.PacketConn
	congestionControl string
	cwnd              int
	bbrProfile        string
}

func NewQUICService(service *Service, congestionControl string, cwnd int, bbrProfile string) *QUICService {
	return &QUICService{
		service:           service,
		congestionControl: congestionControl,
		cwnd:              cwnd,
		bbrProfile:        bbrProfile,
	}
}

func (s *QUICService) Start(ctx context.Context, udpConn net.PacketConn, tlsConfig tls.ServerConfig) error {
	s.udpConn = udpConn
	congestionControlFactory, err := NewCongestionControl(s.congestionControl, s.cwnd, s.bbrProfile, ntp.TimeFuncFromContext(ctx))
	if err != nil {
		return err
	}
	s.h3Server = &http3.Server{
		Handler: s.service,
		ConnContext: func(ctx context.Context, conn *quic.Conn) context.Context {
			conn.SetCongestionControl(congestionControlFactory(conn))
			return ctx
		},
	}
	if err := qtls.ConfigureHTTP3(tlsConfig); err != nil {
		return err
	}
	quicListener, err := qtls.ListenEarly(udpConn, tlsConfig, &quic.Config{
		MaxIdleTimeout:    DefaultSessionTimeout * 2,
		MaxIncomingStreams: 1 << 60,
		Allow0RTT:         true,
	})
	if err != nil {
		return err
	}
	go func() {
		_ = s.h3Server.ServeListener(quicListener)
	}()
	return nil
}

func (s *QUICService) Close() error {
	var errs []error
	if s.h3Server != nil {
		errs = append(errs, s.h3Server.Close())
	}
	if s.udpConn != nil {
		errs = append(errs, s.udpConn.Close())
	}
	return errors.Join(errs...)
}
