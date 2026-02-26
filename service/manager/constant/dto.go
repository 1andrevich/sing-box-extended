package constant

import "time"

type Squad struct {
	ID        int       `json:"id" validate:"required"`
	Name      string    `json:"name" validate:"required"`
	CreatedAt time.Time `json:"created_at" validate:"required"`
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

type SquadCreate struct {
	Name string `json:"name" validate:"required"`
}

type SquadUpdate struct {
	Name string `json:"name" validate:"required"`
}

type Node struct {
	UUID      string    `json:"uuid" validate:"required,uuid4"`
	Name      string    `json:"name" validate:"required"`
	SquadIDs  []int     `json:"squad_ids" validate:"required"`
	CreatedAt time.Time `json:"created_at" validate:"required"`
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

type NodeCreate struct {
	UUID     string `json:"uuid" validate:"required,uuid4"`
	Name     string `json:"name" validate:"required"`
	SquadIDs []int  `json:"squad_ids" validate:"required"`
}

type NodeUpdate struct {
	Name string `json:"name" validate:"required"`
}

type BaseNode struct {
	UUID string `json:"uuid" validate:"required,uuid4"`
	Name string `json:"name" validate:"required"`
}

type User struct {
	ID        int       `json:"id" validate:"required"`
	SquadIDs  []int     `json:"squad_ids" validate:"required"`
	Username  string    `json:"username" validate:"required"`
	Type      string    `json:"type" validate:"required"`
	Inbound   string    `json:"inbound" validate:"required"`
	UUID      string    `json:"uuid" validate:"required"`
	Password  string    `json:"password" validate:"required"`
	Flow      string    `json:"flow" validate:"required"`
	AlterID   int       `json:"alter_id" validate:"required"`
	CreatedAt time.Time `json:"created_at" validate:"required"`
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

type UserCreate struct {
	SquadIDs []int  `json:"squad_ids" validate:"required"`
	Username string `json:"username" validate:"required"`
	Type     string `json:"type" validate:"required,oneof=hysteria hysteria2 trojan tuic vless vmess"`
	Inbound  string `json:"inbound" validate:"required"`
	UUID     string `json:"uuid" validate:"omitempty,uuid4"`
	Password string `json:"password" validate:"omitempty"`
	Flow     string `json:"flow" validate:"omitempty"`
	AlterID  int    `json:"alter_id" validate:"omitempty"`
}

type UserUpdate struct {
	UUID     string `json:"uuid" validate:"omitempty,uuid4"`
	Password string `json:"password" validate:"omitempty"`
	Flow     string `json:"flow" validate:"omitempty"`
	AlterID  int    `json:"alter_id" validate:"omitempty"`
}

type BaseUser struct {
	UUID     string `json:"uuid" validate:"omitempty,uuid4"`
	Password string `json:"password" validate:"omitempty"`
	Flow     string `json:"flow" validate:"omitempty"`
	AlterID  int    `json:"alter_id" validate:"omitempty"`
}

type ConnectionLimiter struct {
	ID             int       `json:"id" validate:"required"`
	SquadIDs       []int     `json:"squad_ids" validate:"required"`
	Username       string    `json:"username" validate:"required"`
	Outbound       string    `json:"outbound" validate:"required"`
	Strategy       string    `json:"strategy" validate:"required,oneof=connection"`
	ConnectionType string    `json:"connection_type" validate:"omitempty,oneof=hwid mux ip"`
	LockType       string    `json:"lock_type" validate:"omitempty,oneof=manager"`
	Count          uint32    `json:"count" validate:"required"`
	CreatedAt      time.Time `json:"created_at" validate:"required"`
	UpdatedAt      time.Time `json:"updated_at" validate:"required"`
}

type ConnectionLimiterCreate struct {
	SquadIDs       []int  `json:"squad_ids" validate:"required"`
	Username       string `json:"username" validate:"required"`
	Outbound       string `json:"outbound" validate:"required"`
	Strategy       string `json:"strategy" validate:"required,oneof=connection"`
	ConnectionType string `json:"type" validate:"omitempty,oneof=hwid mux ip"`
	LockType       string `json:"lock_type" validate:"omitempty,oneof=manager"`
	Count          uint32 `json:"count" validate:"required"`
}

type ConnectionLimiterUpdate struct {
	Username       string `json:"username" validate:"required"`
	Outbound       string `json:"outbound" validate:"required"`
	Strategy       string `json:"strategy" validate:"required,oneof=connection"`
	ConnectionType string `json:"type" validate:"omitempty,oneof=hwid mux ip"`
	LockType       string `json:"lock_type" validate:"omitempty,oneof=manager"`
	Count          uint32 `json:"count" validate:"required"`
}

type BaseConnectionLimiter struct {
	Username       string `json:"username" validate:"required"`
	Outbound       string `json:"outbound" validate:"required"`
	Strategy       string `json:"strategy" validate:"required,oneof=connection"`
	ConnectionType string `json:"type" validate:"omitempty,oneof=hwid mux ip"`
	LockType       string `json:"lock_type" validate:"omitempty,oneof=manager"`
	Count          uint32 `json:"count" validate:"required"`
}

type BandwidthLimiter struct {
	ID             int       `json:"id" validate:"required"`
	SquadIDs       []int     `json:"squad_ids" validate:"required"`
	Username       string    `json:"username" validate:"required"`
	Outbound       string    `json:"outbound" validate:"required"`
	Strategy       string    `json:"strategy" validate:"required"`
	Mode           string    `json:"mode" validate:"required"`
	ConnectionType string    `json:"connection_type" validate:"omitempty"`
	Speed          string    `json:"speed" validate:"required"`
	RawSpeed       uint64    `json:"raw_speed" validate:"required"`
	CreatedAt      time.Time `json:"created_at" validate:"required"`
	UpdatedAt      time.Time `json:"updated_at" validate:"required"`
}

type BandwidthLimiterCreate struct {
	SquadIDs       []int  `json:"squad_ids" validate:"required"`
	Username       string `json:"username" validate:"required"`
	Outbound       string `json:"outbound" validate:"required"`
	Strategy       string `json:"strategy" validate:"required,oneof=global connection"`
	Mode           string `json:"mode" validate:"required"`
	ConnectionType string `json:"connection_type" validate:"omitempty"`
	Speed          string `json:"speed" validate:"required"`
}

type BandwidthLimiterUpdate struct {
	Username       string `json:"username" validate:"required"`
	Outbound       string `json:"outbound" validate:"required"`
	Strategy       string `json:"strategy" validate:"required,oneof=global connection"`
	Mode           string `json:"mode" validate:"required"`
	ConnectionType string `json:"connection_type" validate:"omitempty"`
	Speed          string `json:"speed" validate:"required"`
}

type BaseBandwidthLimiter struct {
	Username       string `json:"username" validate:"required"`
	Outbound       string `json:"outbound" validate:"required"`
	Strategy       string `json:"strategy" validate:"required,oneof=global connection"`
	Mode           string `json:"mode" validate:"required"`
	ConnectionType string `json:"connection_type" validate:"omitempty"`
	Speed          string `json:"speed" validate:"required"`
	RawSpeed       uint64 `json:"raw_speed" validate:"required"`
}
