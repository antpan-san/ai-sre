package models

import (
	"time"

	"github.com/google/uuid"
)

// K8sBundleInvite 保存控制台生成的离线安装配置，供目标机仅凭资源 ID + token 拉取 zip（无需登录）。
type K8sBundleInvite struct {
	BaseModel
	RequestPayload  JSONB     `gorm:"type:jsonb;not null" json:"request_payload"`
	DownloadToken   string    `gorm:"size:64;not null" json:"-"`
	ExpiresAt       time.Time `gorm:"not null;index" json:"expires_at"`
	CreatedByUserID uuid.UUID `gorm:"type:uuid;index" json:"created_by_user_id"`
}

func (K8sBundleInvite) TableName() string {
	return "k8s_bundle_invites"
}
