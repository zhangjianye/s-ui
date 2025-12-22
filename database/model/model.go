package model

import "encoding/json"

type Setting struct {
	Id    uint   `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Key   string `json:"key" form:"key"`
	Value string `json:"value" form:"value"`
}

type Tls struct {
	Id     uint            `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Name   string          `json:"name" form:"name"`
	Server json.RawMessage `json:"server" form:"server"`
	Client json.RawMessage `json:"client" form:"client"`
}

type User struct {
	Id         uint   `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Username   string `json:"username" form:"username"`
	Password   string `json:"password" form:"password"`
	LastLogins string `json:"lastLogin"`
}

type Client struct {
	Id       uint            `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Enable   bool            `json:"enable" form:"enable"`
	Name     string          `json:"name" form:"name"`
	Config   json.RawMessage `json:"config,omitempty" form:"config"`
	Inbounds json.RawMessage `json:"inbounds" form:"inbounds"`
	Links    json.RawMessage `json:"links,omitempty" form:"links"`
	Volume   int64           `json:"volume" form:"volume"`
	Expiry   int64           `json:"expiry" form:"expiry"`
	Down     int64           `json:"down" form:"down"`
	Up       int64           `json:"up" form:"up"`
	Desc     string          `json:"desc" form:"desc"`
	Group    string          `json:"group" form:"group"`
	// UAP 扩展字段
	UUID                 string `json:"uuid" form:"uuid" gorm:"unique"`
	IsPremium            bool   `json:"isPremium" form:"isPremium" gorm:"default:false"`
	TimeLimit            int64  `json:"timeLimit" form:"timeLimit" gorm:"default:0"`
	TimeUsed             int64  `json:"timeUsed" form:"timeUsed" gorm:"default:0"`
	TimeResetStrategy    string `json:"timeResetStrategy" form:"timeResetStrategy" gorm:"default:'no_reset'"`
	TimeResetAt          int64  `json:"timeResetAt" form:"timeResetAt" gorm:"default:0"`
	TrafficResetStrategy string `json:"trafficResetStrategy" form:"trafficResetStrategy" gorm:"default:'no_reset'"`
	TrafficResetAt       int64  `json:"trafficResetAt" form:"trafficResetAt" gorm:"default:0"`
	SpeedLimit           int    `json:"speedLimit" form:"speedLimit" gorm:"default:0"`
	DeviceLimit          int    `json:"deviceLimit" form:"deviceLimit" gorm:"default:0"`
}

type Stats struct {
	Id        uint64 `json:"id" gorm:"primaryKey;autoIncrement"`
	DateTime  int64  `json:"dateTime"`
	Resource  string `json:"resource"`
	Tag       string `json:"tag"`
	Direction bool   `json:"direction"`
	Traffic   int64  `json:"traffic"`
	NodeId    string `json:"nodeId" gorm:"index;default:'local'"`
}

type Changes struct {
	Id       uint64          `json:"id" gorm:"primaryKey;autoIncrement"`
	DateTime int64           `json:"dateTime"`
	Actor    string          `json:"actor"`
	Key      string          `json:"key"`
	Action   string          `json:"action"`
	Obj      json.RawMessage `json:"obj"`
}

type Tokens struct {
	Id     uint   `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Desc   string `json:"desc" form:"desc"`
	Token  string `json:"token" form:"token"`
	Expiry int64  `json:"expiry" form:"expiry"`
	UserId uint   `json:"userId" form:"userId"`
	User   *User  `json:"user" gorm:"foreignKey:UserId;references:Id"`
}
