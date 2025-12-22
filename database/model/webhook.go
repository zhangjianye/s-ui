package model

// WebhookConfig Webhook 配置 (UAP 回调)
type WebhookConfig struct {
	Id             uint   `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	CallbackURL    string `json:"callbackUrl" form:"callbackUrl"`
	CallbackSecret string `json:"callbackSecret" form:"callbackSecret"`
	Enable         bool   `json:"enable" form:"enable" gorm:"default:true"`
}

// ApiKey API Key (UAP 认证)
type ApiKey struct {
	Id        uint   `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Key       string `json:"key" form:"key" gorm:"unique;not null"`
	Name      string `json:"name" form:"name"`
	Enable    bool   `json:"enable" form:"enable" gorm:"default:true"`
	CreatedAt int64  `json:"createdAt" gorm:"autoCreateTime"`
}
