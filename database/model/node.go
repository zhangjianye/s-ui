package model

import "encoding/json"

// NodeToken 节点邀请码
type NodeToken struct {
	Id        uint   `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Token     string `json:"token" form:"token" gorm:"unique;not null"`
	Name      string `json:"name" form:"name"`
	Used      bool   `json:"used" form:"used" gorm:"default:false"`
	UsedBy    string `json:"usedBy" form:"usedBy"`
	ExpiresAt int64  `json:"expiresAt" form:"expiresAt"`
	CreatedAt int64  `json:"createdAt" gorm:"autoCreateTime"`
}

// Node 节点注册信息
type Node struct {
	Id           uint            `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	NodeId       string          `json:"nodeId" form:"nodeId" gorm:"unique;not null"`
	Name         string          `json:"name" form:"name" gorm:"not null"`
	Address      string          `json:"address" form:"address"`
	ExternalHost string          `json:"externalHost" form:"externalHost"`
	ExternalPort int             `json:"externalPort" form:"externalPort" gorm:"default:0"`
	Token        string          `json:"token" form:"token" gorm:"not null"`
	Enable       bool            `json:"enable" form:"enable" gorm:"default:true"`
	Status       string          `json:"status" form:"status" gorm:"default:'offline'"`
	LastSeen     int64           `json:"lastSeen" form:"lastSeen"`
	LastSync     int64           `json:"lastSync" form:"lastSync"`
	Version      string          `json:"version" form:"version"`
	SystemInfo   json.RawMessage `json:"systemInfo" form:"systemInfo" gorm:"type:text"`
	CreatedAt    int64           `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt    int64           `json:"updatedAt" gorm:"autoUpdateTime"`
	// UAP API 所需字段
	Country   string `json:"country" form:"country"`
	City      string `json:"city" form:"city"`
	Flag      string `json:"flag" form:"flag"`
	IsPremium bool   `json:"isPremium" form:"isPremium" gorm:"default:false"`
	Latency   int    `json:"latency" form:"latency" gorm:"default:0"`
}

// NodeStats 节点统计快照
type NodeStats struct {
	Id          uint64  `json:"id" gorm:"primaryKey;autoIncrement"`
	NodeId      string  `json:"nodeId" gorm:"index;not null"`
	DateTime    int64   `json:"dateTime" gorm:"index;not null"`
	CPU         float64 `json:"cpu"`
	Memory      float64 `json:"memory"`
	Connections int     `json:"connections"`
	Upload      int64   `json:"upload"`
	Download    int64   `json:"download"`
}

// ClientOnline 客户端在线状态
type ClientOnline struct {
	Id          uint64 `json:"id" gorm:"primaryKey;autoIncrement"`
	ClientName  string `json:"clientName" gorm:"index;not null"`
	NodeId      string `json:"nodeId" gorm:"index;not null"`
	InboundTag  string `json:"inboundTag"`
	SourceIP    string `json:"sourceIP"`
	ConnectedAt int64  `json:"connectedAt"`
	LastSeen    int64  `json:"lastSeen" gorm:"not null"`
}
