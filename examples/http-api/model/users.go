package model

import (
	"context"
	"time"

	"github.com/goupter/goupter/pkg/model"
	"gorm.io/gorm"
)

// Users users表模型
type Users struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Username  string    `gorm:"column:username" json:"username"`
	Password  string    `gorm:"column:password" json:"password"`
	Email     string    `gorm:"column:email" json:"email"`
	Role      string    `gorm:"column:role" json:"role"`
	Status    int8      `gorm:"column:status" json:"status"` // 1=active, 0=inactive
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName 表名
func (m *Users) TableName() string {
	return "users"
}

// UsersModel Users模型（嵌入泛型BaseModel）
type UsersModel struct {
	*model.BaseModel[Users]
}

// NewUsersModel 创建Users模型
func NewUsersModel(db *gorm.DB) *UsersModel {
	return &UsersModel{
		BaseModel: model.NewBaseModel[Users](db),
	}
}

// FindByUsername 根据用户名查找用户
func (m *UsersModel) FindByUsername(ctx context.Context, username string) (*Users, error) {
	return m.FindOne(ctx, map[string]any{"username": username})
}
