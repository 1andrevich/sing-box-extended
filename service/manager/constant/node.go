package constant

type ConnectedNode interface {
	UpdateUser(user User)
	UpdateUsers(users []User)
	DeleteUser(user User)

	UpdateConnectionLimiter(limiter ConnectionLimiter)
	UpdateConnectionLimiters(limiter []ConnectionLimiter)
	DeleteConnectionLimiter(limiter ConnectionLimiter)

	UpdateBandwidthLimiter(limiter BandwidthLimiter)
	UpdateBandwidthLimiters(limiter []BandwidthLimiter)
	DeleteBandwidthLimiter(limiter BandwidthLimiter)

	IsLocal() bool
	IsOnline() bool

	Close() error
}
