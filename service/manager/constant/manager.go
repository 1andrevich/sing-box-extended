package constant

type NodeManager interface {
	AddNode(id string, node ConnectedNode) error
	AcquireLock(limiterId int, id string) (string, error)
	RefreshLock(limiterId int, id string, handleId string) error
	ReleaseLock(limiterId int, id string, handleId string) error
}

type Manager interface {
	NodeManager

	CreateSquad(user SquadCreate) (Squad, error)
	GetSquads(filters map[string][]string) ([]Squad, error)
	GetSquadsCount(filters map[string][]string) (int, error)
	GetSquad(id int) (Squad, error)
	UpdateSquad(id int, user SquadUpdate) (Squad, error)
	DeleteSquad(id int) (Squad, error)

	CreateNode(node NodeCreate) (Node, error)
	GetNodes(filters map[string][]string) ([]Node, error)
	GetNodesCount(filters map[string][]string) (int, error)
	GetNode(uuid string) (Node, error)
	GetNodeStatus(uuid string) string
	UpdateNode(uuid string, node NodeUpdate) (Node, error)
	DeleteNode(uuid string) (Node, error)

	CreateUser(user UserCreate) (User, error)
	GetUsers(filters map[string][]string) ([]User, error)
	GetUsersCount(filters map[string][]string) (int, error)
	GetUser(id int) (User, error)
	UpdateUser(id int, user UserUpdate) (User, error)
	DeleteUser(id int) (User, error)

	CreateBandwidthLimiter(limiter BandwidthLimiterCreate) (BandwidthLimiter, error)
	GetBandwidthLimiters(filters map[string][]string) ([]BandwidthLimiter, error)
	GetBandwidthLimitersCount(filters map[string][]string) (int, error)
	GetBandwidthLimiter(id int) (BandwidthLimiter, error)
	UpdateBandwidthLimiter(id int, limiter BandwidthLimiterUpdate) (BandwidthLimiter, error)
	DeleteBandwidthLimiter(id int) (BandwidthLimiter, error)

	CreateConnectionLimiter(limiter ConnectionLimiterCreate) (ConnectionLimiter, error)
	GetConnectionLimiters(filters map[string][]string) ([]ConnectionLimiter, error)
	GetConnectionLimitersCount(filters map[string][]string) (int, error)
	GetConnectionLimiter(id int) (ConnectionLimiter, error)
	UpdateConnectionLimiter(id int, limiter ConnectionLimiterUpdate) (ConnectionLimiter, error)
	DeleteConnectionLimiter(id int) (ConnectionLimiter, error)
}
