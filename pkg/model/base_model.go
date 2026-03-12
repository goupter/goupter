package model

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm"
)

var (
	ErrNotFound  = errors.New("record not found")
	ErrInvalidID = errors.New("invalid id")
)

// BaseModel 泛型基础模型
type BaseModel[T any] struct {
	db              *gorm.DB
	geometryColumns []string // cached geometry columns
	selectClause    string   // cached select clause with ST_AsText
}

// NewBaseModel 创建泛型基础模型
func NewBaseModel[T any](db *gorm.DB) *BaseModel[T] {
	m := &BaseModel[T]{db: db}
	m.initGeometryColumns()
	return m
}

// initGeometryColumns initializes geometry columns from the model
func (m *BaseModel[T]) initGeometryColumns() {
	var t T
	// Check if T implements GeometryColumnsProvider
	if provider, ok := any(&t).(GeometryColumnsProvider); ok {
		m.geometryColumns = provider.GeometryColumns()
		m.selectClause = m.buildSelectClause()
	}
}

// buildSelectClause builds SELECT clause with ST_AsText for geometry columns
func (m *BaseModel[T]) buildSelectClause() string {
	if len(m.geometryColumns) == 0 {
		return ""
	}

	var t T
	typ := reflect.TypeOf(t)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// Build geometry column set for quick lookup
	geomSet := make(map[string]bool)
	for _, col := range m.geometryColumns {
		geomSet[col] = true
	}

	// Build select clause
	var parts []string
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		// Get column name from gorm tag
		gormTag := field.Tag.Get("gorm")
		colName := parseColumnName(gormTag)
		if colName == "" {
			colName = toSnakeCase(field.Name)
		}

		if geomSet[colName] {
			// Use ST_AsText for geometry columns with axis-order=long-lat for SRID 4326
			parts = append(parts, fmt.Sprintf("ST_AsText(`%s`, 'axis-order=long-lat') AS `%s`", colName, colName))
		} else {
			parts = append(parts, fmt.Sprintf("`%s`", colName))
		}
	}

	return strings.Join(parts, ", ")
}

// parseColumnName extracts column name from gorm tag
func parseColumnName(tag string) string {
	parts := strings.Split(tag, ";")
	for _, part := range parts {
		if strings.HasPrefix(part, "column:") {
			return strings.TrimPrefix(part, "column:")
		}
	}
	return ""
}

// toSnakeCase converts CamelCase to snake_case
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		if r >= 'A' && r <= 'Z' {
			result.WriteByte(byte(r + 32))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// applyGeometrySelect applies ST_AsText select clause if model has geometry columns
func (m *BaseModel[T]) applyGeometrySelect(db *gorm.DB) *gorm.DB {
	if m.selectClause != "" {
		return db.Select(m.selectClause)
	}
	return db
}

// Insert 插入数据
func (m *BaseModel[T]) Insert(ctx context.Context, tx *gorm.DB, data *T) error {
	return m.getDB(tx).WithContext(ctx).Create(data).Error
}

// FindOne 根据条件查询单条
func (m *BaseModel[T]) FindOne(ctx context.Context, condition map[string]any) (*T, error) {
	var result T
	db := m.applyGeometrySelect(m.db.WithContext(ctx))
	err := db.Where(condition).First(&result).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &result, nil
}

// Update 更新数据
func (m *BaseModel[T]) Update(ctx context.Context, tx *gorm.DB, data *T) error {
	return m.getDB(tx).WithContext(ctx).Save(data).Error
}

// UpdateColumns 更新指定列
func (m *BaseModel[T]) UpdateColumns(ctx context.Context, tx *gorm.DB, condition map[string]any, columns map[string]any) error {
	return m.getDB(tx).WithContext(ctx).Model(new(T)).Where(condition).Updates(columns).Error
}

// FindAll 查询所有记录
func (m *BaseModel[T]) FindAll(ctx context.Context, orderBy string, query string, args ...any) ([]*T, error) {
	results, _, err := m.FindPage(ctx, 0, 0, orderBy, query, args...)
	return results, err
}

// FindCount 查询记录数
func (m *BaseModel[T]) FindCount(ctx context.Context, query string, args ...any) (int64, error) {
	var count int64
	db := m.db.WithContext(ctx).Model(new(T))
	if query != "" {
		db = db.Where(query, args...)
	}
	return count, db.Count(&count).Error
}

// FindPage 分页查询
func (m *BaseModel[T]) FindPage(ctx context.Context, page, pageSize int, orderBy string, query string, args ...any) ([]*T, int64, error) {
	var results []*T
	var total int64

	db := m.applyGeometrySelect(m.db.WithContext(ctx).Model(new(T)))
	if query != "" {
		db = db.Where(query, args...)
	}

	if page > 0 && pageSize > 0 {
		if err := db.Count(&total).Error; err != nil {
			return nil, 0, err
		}
		if orderBy != "" {
			db = db.Order(orderBy)
		}
		offset := (page - 1) * pageSize
		return results, total, db.Offset(offset).Limit(pageSize).Find(&results).Error
	}

	if orderBy != "" {
		db = db.Order(orderBy)
	}
	err := db.Find(&results).Error
	return results, int64(len(results)), err
}

// Delete 删除数据
func (m *BaseModel[T]) Delete(ctx context.Context, tx *gorm.DB, condition map[string]any) error {
	return m.getDB(tx).WithContext(ctx).Where(condition).Delete(new(T)).Error
}

// Transaction 事务
func (m *BaseModel[T]) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return m.db.WithContext(ctx).Transaction(fn)
}

// Exec 执行自定义SQL
func (m *BaseModel[T]) Exec(ctx context.Context, tx *gorm.DB, sql string, args ...any) error {
	return m.getDB(tx).WithContext(ctx).Exec(sql, args...).Error
}

// Query 执行自定义查询
func (m *BaseModel[T]) Query(ctx context.Context, dest any, sql string, args ...any) error {
	return m.db.WithContext(ctx).Raw(sql, args...).Scan(dest).Error
}

// DB 获取底层DB
func (m *BaseModel[T]) DB() *gorm.DB {
	return m.db
}

func (m *BaseModel[T]) getDB(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return m.db
}
