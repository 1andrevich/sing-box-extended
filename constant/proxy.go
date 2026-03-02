package constant

const (
	TypeTun               = "tun"
	TypeRedirect          = "redirect"
	TypeTProxy            = "tproxy"
	TypeDirect            = "direct"
	TypeBlock             = "block"
	TypeDNS               = "dns"
	TypeSOCKS             = "socks"
	TypeHTTP              = "http"
	TypeMixed             = "mixed"
	TypeShadowsocks       = "shadowsocks"
	TypeVMess             = "vmess"
	TypeTrojan            = "trojan"
	TypeNaive             = "naive"
	TypeWireGuard         = "wireguard"
	TypeWARP              = "warp"
	TypeHysteria          = "hysteria"
	TypeTor               = "tor"
	TypeSSH               = "ssh"
	TypeShadowTLS         = "shadowtls"
	TypeMieru             = "mieru"
	TypeAnyTLS            = "anytls"
	TypeShadowsocksR      = "shadowsocksr"
	TypeVLESS             = "vless"
	TypeTUIC              = "tuic"
	TypeHysteria2         = "hysteria2"
	TypeBond              = "bond"
	TypeTunnelServer      = "tunnel-server"
	TypeTunnelClient      = "tunnel-client"
	TypeTailscale         = "tailscale"
	TypeConnectionLimiter = "connection-limiter"
	TypeBandwidthLimiter  = "bandwidth-limiter"
	TypeTrafficLimiter    = "traffic-limiter"
	TypeAdminPanel        = "admin-panel"
	TypeNodeManagerServer = "node-manager-server"
	TypeNodeManagerClient = "node-manager-client"
	TypeDERP              = "derp"
	TypeManager           = "manager"
	TypeNode              = "node"
	TypeResolved          = "resolved"
	TypeSSMAPI            = "ssm-api"
)

const (
	TypeFailover = "failover"
	TypeSelector = "selector"
	TypeURLTest  = "urltest"
)

func ProxyDisplayName(proxyType string) string {
	switch proxyType {
	case TypeTun:
		return "TUN"
	case TypeRedirect:
		return "Redirect"
	case TypeTProxy:
		return "TProxy"
	case TypeDirect:
		return "Direct"
	case TypeBlock:
		return "Block"
	case TypeDNS:
		return "DNS"
	case TypeSOCKS:
		return "SOCKS"
	case TypeHTTP:
		return "HTTP"
	case TypeMixed:
		return "Mixed"
	case TypeShadowsocks:
		return "Shadowsocks"
	case TypeVMess:
		return "VMess"
	case TypeTrojan:
		return "Trojan"
	case TypeNaive:
		return "Naive"
	case TypeWireGuard:
		return "WireGuard"
	case TypeWARP:
		return "WARP"
	case TypeHysteria:
		return "Hysteria"
	case TypeTor:
		return "Tor"
	case TypeSSH:
		return "SSH"
	case TypeShadowTLS:
		return "ShadowTLS"
	case TypeShadowsocksR:
		return "ShadowsocksR"
	case TypeVLESS:
		return "VLESS"
	case TypeTUIC:
		return "TUIC"
	case TypeHysteria2:
		return "Hysteria2"
	case TypeBond:
		return "Bond"
	case TypeMieru:
		return "Mieru"
	case TypeAnyTLS:
		return "AnyTLS"
	case TypeFailover:
		return "Failover"
	case TypeSelector:
		return "Selector"
	case TypeURLTest:
		return "URLTest"
	case TypeTunnelClient:
		return "Tunnel client"
	case TypeTunnelServer:
		return "Tunnel server"
	default:
		return "Unknown"
	}
}
