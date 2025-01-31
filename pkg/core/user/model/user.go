package model

import (
	"gorm.io/gorm"
	"time"
)

type User struct {
	ID           int64          `gorm:"primaryKey;autoIncrement"`
	Username     string         `gorm:"type:varchar(100);uniqueIndex;not null"`
	Email        string         `gorm:"type:varchar(255);uniqueIndex;not null"`
	PasswordHash string         `gorm:"type:varchar(255);not null"`
	IsActive     bool           `gorm:"default:true;index"`
	Version      int            `gorm:"default:1;not null"` // 新增乐观锁配置
	CreatedAt    time.Time      `gorm:"index;autoCreateTime"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `gorm:"index"` // 软删除标记
}

// TableName 定义映射表名
func (User) TableName() string {
	return "base_users" // 更清晰的表名
}

func AutoMigrate(db *gorm.DB) error {
	return db.Set("gorm:table_options", "COMMENT='用户基础表'").
		AutoMigrate(&User{})
}
