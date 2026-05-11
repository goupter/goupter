package model

import (
	"context"
	"time"

	"github.com/goupter/goupter/pkg/model"
	"gorm.io/gorm"
)

// Articles articles表模型
type Articles struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Title     string    `gorm:"column:title" json:"title"`
	Content   string    `gorm:"column:content" json:"content"`
	AuthorID  int64     `gorm:"column:author_id" json:"author_id"`
	Status    int8      `gorm:"column:status" json:"status"` // 1=published, 0=draft
	ViewCount int32     `gorm:"column:view_count" json:"view_count"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName 表名
func (m *Articles) TableName() string {
	return "articles"
}

// ArticlesModel Articles模型（嵌入泛型BaseModel）
type ArticlesModel struct {
	*model.BaseModel[Articles]
}

// NewArticlesModel 创建Articles模型
func NewArticlesModel(db *gorm.DB) *ArticlesModel {
	return &ArticlesModel{
		BaseModel: model.NewBaseModel[Articles](db),
	}
}

// FindByID 根据ID查找文章
func (m *ArticlesModel) FindByID(ctx context.Context, id int64) (*Articles, error) {
	return m.FindOne(ctx, map[string]any{"id": id})
}

// IncrViewCount 增加浏览次数
func (m *ArticlesModel) IncrViewCount(ctx context.Context, id int64) error {
	return m.UpdateColumns(ctx, nil, map[string]any{"id": id}, map[string]any{
		"view_count": gorm.Expr("view_count + 1"),
	})
}
