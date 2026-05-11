package gen

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"

	_ "github.com/go-sql-driver/mysql"
)

// ErrFileExists indicates the file already exists and should be skipped
var ErrFileExists = errors.New("file already exists")

// ModelConfig 模型生成配置
type ModelConfig struct {
	DSN         string // 数据库连接字符串
	Tables      string // 要生成的表，逗号分隔
	OutDir      string // 输出目录
	PackageName string // 包名
	TrimPrefix  string // 过滤表名前缀
	WithQuery   bool   // 是否生成查询方法
}

// ModelGenerator 模型生成器
type ModelGenerator struct {
	config ModelConfig
	db     *sql.DB
}

// TableInfo 表信息
type TableInfo struct {
	Name            string // 显示名（过滤前缀后）
	RawName         string // 原始表名（用于 TableName()）
	Comment         string
	Columns         []ColumnInfo
	PrimaryKey      string
	PackageName     string
	StructName      string
	NeedTime        bool     // 是否需要导入 time 包
	NeedSQL         bool     // 是否需要导入 database/sql 包
	GeometryColumns []string // geometry 类型的列名
}

// ColumnInfo 列信息
type ColumnInfo struct {
	Name      string
	Type      string
	GoType    string
	Comment   string
	Nullable  bool
	IsPrimary bool
	Default   sql.NullString
	Extra     string
	Tag       string
	FieldName string
}

// NewModelGenerator 创建模型生成器
func NewModelGenerator(config ModelConfig) (*ModelGenerator, error) {
	db, err := sql.Open("mysql", config.DSN)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping数据库失败: %w", err)
	}

	return &ModelGenerator{
		config: config,
		db:     db,
	}, nil
}

// Close 关闭数据库连接
func (g *ModelGenerator) Close() error {
	if g.db != nil {
		return g.db.Close()
	}
	return nil
}

// Generate 生成模型
func (g *ModelGenerator) Generate() error {
	// 创建输出目录
	if err := os.MkdirAll(g.config.OutDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 获取要生成的表
	tables, err := g.getTables()
	if err != nil {
		return err
	}

	if len(tables) == 0 {
		return fmt.Errorf("没有找到要生成的表")
	}

	// 为每个表生成模型
	for _, tableName := range tables {
		tableInfo, err := g.getTableInfo(tableName)
		if err != nil {
			return fmt.Errorf("获取表 %s 信息失败: %w", tableName, err)
		}

		if err := g.generateModelFile(tableInfo); err != nil {
			if errors.Is(err, ErrFileExists) {
				fmt.Printf("  ⏭ 跳过 %s.go（已存在）\n", tableName)
				continue
			}
			return fmt.Errorf("生成表 %s 模型失败: %w", tableName, err)
		}

		fmt.Printf("  ✓ 生成 %s.go\n", tableName)
	}

	return nil
}

// GetTables 获取要生成的表列表（导出方法）
func (g *ModelGenerator) GetTables() ([]string, error) {
	return g.getTables()
}

// GetTableInfo 获取表信息（导出方法）
func (g *ModelGenerator) GetTableInfo(tableName string) (*TableInfo, error) {
	return g.getTableInfo(tableName)
}

// getTables 获取要生成的表列表
func (g *ModelGenerator) getTables() ([]string, error) {
	if g.config.Tables != "" {
		// 使用指定的表
		tables := strings.Split(g.config.Tables, ",")
		for i := range tables {
			tables[i] = strings.TrimSpace(tables[i])
		}
		return tables, nil
	}

	// 获取所有表
	rows, err := g.db.Query("SHOW TABLES")
	if err != nil {
		return nil, fmt.Errorf("查询表列表失败: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}

	return tables, nil
}

// getTableInfo 获取表信息
func (g *ModelGenerator) getTableInfo(tableName string) (*TableInfo, error) {
	// 应用前缀过滤
	displayName := tableName
	if g.config.TrimPrefix != "" {
		displayName = strings.TrimPrefix(tableName, g.config.TrimPrefix)
		displayName = strings.TrimPrefix(displayName, "_")
	}

	info := &TableInfo{
		Name:        displayName,
		RawName:     tableName, // 保留原始表名
		PackageName: g.config.PackageName,
		StructName:  toCamelCase(displayName, true),
	}

	// 获取表注释
	var tableComment sql.NullString
	err := g.db.QueryRow(`
		SELECT TABLE_COMMENT
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
	`, tableName).Scan(&tableComment)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	info.Comment = tableComment.String

	// 获取列信息
	rows, err := g.db.Query(`
		SELECT
			COLUMN_NAME,
			DATA_TYPE,
			COLUMN_COMMENT,
			IS_NULLABLE,
			COLUMN_KEY,
			COLUMN_DEFAULT,
			EXTRA
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION
	`, tableName)
	if err != nil {
		return nil, fmt.Errorf("查询列信息失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var col ColumnInfo
		var nullable, columnKey string
		var colDefault sql.NullString

		if err := rows.Scan(&col.Name, &col.Type, &col.Comment, &nullable, &columnKey, &colDefault, &col.Extra); err != nil {
			return nil, err
		}

		col.Nullable = nullable == "YES"
		col.IsPrimary = columnKey == "PRI"
		col.Default = colDefault
		col.GoType = mysqlTypeToGoType(col.Type, col.Nullable)
		col.FieldName = toCamelCase(col.Name, true)
		col.Tag = buildGormTag(col)

		if col.IsPrimary {
			info.PrimaryKey = col.Name
		}

		// 检测是否需要导入特定包
		if col.GoType == "time.Time" {
			info.NeedTime = true
		}
		if strings.HasPrefix(col.GoType, "sql.Null") {
			info.NeedSQL = true
		}
		// 记录 geometry 类型的列
		if col.GoType == "model.Geometry" {
			info.GeometryColumns = append(info.GeometryColumns, col.Name)
		}

		info.Columns = append(info.Columns, col)
	}

	return info, nil
}

// generateModelFile 生成模型文件（使用泛型BaseModel）
func (g *ModelGenerator) generateModelFile(info *TableInfo) error {
	tmpl := `package {{.PackageName}}

import (
{{- if .NeedSQL}}
	"database/sql"
{{- end}}
{{- if .NeedTime}}
	"time"
{{- end}}

	"github.com/goupter/goupter/pkg/model"
	"gorm.io/gorm"
)

{{if .Comment}}// {{.StructName}} {{.Comment}}{{else}}// {{.StructName}} {{.Name}}表模型{{end}}
type {{.StructName}} struct {
{{- range .Columns}}
	{{.FieldName}} {{.GoType}} {{.Tag}}{{if .Comment}} // {{.Comment}}{{end}}
{{- end}}
}

// TableName 表名
func (m *{{.StructName}}) TableName() string {
	return "{{.RawName}}"
}
{{if .GeometryColumns}}
// GeometryColumns returns the list of geometry columns
// These columns will be automatically converted using ST_AsText() in queries
func (m *{{.StructName}}) GeometryColumns() []string {
	return []string{ {{- range $i, $col := .GeometryColumns}}{{if $i}}, {{end}}"{{$col}}"{{- end}} }
}
{{end}}
// {{.StructName}}Model {{.StructName}}模型（嵌入泛型BaseModel）
type {{.StructName}}Model struct {
	*model.BaseModel[{{.StructName}}]
}

// New{{.StructName}}Model 创建{{.StructName}}模型
func New{{.StructName}}Model(db *gorm.DB) *{{.StructName}}Model {
	return &{{.StructName}}Model{
		BaseModel: model.NewBaseModel[{{.StructName}}](db),
	}
}
`

	t, err := template.New("model").Parse(tmpl)
	if err != nil {
		return err
	}

	filename := filepath.Join(g.config.OutDir, info.Name+".go")

	// Skip if file already exists
	if _, err := os.Stat(filename); err == nil {
		return ErrFileExists
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return t.Execute(f, info)
}

// toCamelCase 转换为驼峰命名
func toCamelCase(s string, capitalizeFirst bool) string {
	if s == "" {
		return s
	}

	// 处理常见缩写
	commonInitialisms := map[string]string{
		"id":   "ID",
		"url":  "URL",
		"uri":  "URI",
		"api":  "API",
		"http": "HTTP",
		"json": "JSON",
		"xml":  "XML",
		"html": "HTML",
		"ip":   "IP",
		"sql":  "SQL",
		"ssh":  "SSH",
		"tcp":  "TCP",
		"udp":  "UDP",
		"uuid": "UUID",
	}

	parts := strings.Split(s, "_")
	for i, part := range parts {
		if upper, ok := commonInitialisms[strings.ToLower(part)]; ok {
			parts[i] = upper
		} else if i > 0 || capitalizeFirst {
			parts[i] = capitalize(part)
		}
	}

	result := strings.Join(parts, "")
	if !capitalizeFirst && len(result) > 0 {
		runes := []rune(result)
		runes[0] = unicode.ToLower(runes[0])
		result = string(runes)
	}

	return result
}

// capitalize 首字母大写
func capitalize(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// mysqlTypeToGoType MySQL类型转Go类型
func mysqlTypeToGoType(mysqlType string, nullable bool) string {
	mysqlType = strings.ToLower(mysqlType)

	typeMap := map[string]string{
		"tinyint":    "int8",
		"smallint":   "int16",
		"mediumint":  "int32",
		"int":        "int32",
		"integer":    "int32",
		"bigint":     "int64",
		"float":      "float32",
		"double":     "float64",
		"decimal":    "float64",
		"numeric":    "float64",
		"char":       "string",
		"varchar":    "string",
		"text":       "string",
		"tinytext":   "string",
		"mediumtext": "string",
		"longtext":   "string",
		"blob":       "[]byte",
		"tinyblob":   "[]byte",
		"mediumblob": "[]byte",
		"longblob":   "[]byte",
		"binary":     "[]byte",
		"varbinary":  "[]byte",
		"date":       "time.Time",
		"datetime":   "time.Time",
		"timestamp":  "time.Time",
		"time":       "string",
		"year":       "int16",
		"bit":        "[]byte",
		"bool":       "bool",
		"boolean":    "bool",
		"json":       "string",
		"enum":       "string",
		"set":        "string",
		// Spatial types (stored as WKT string via ST_AsText)
		"geometry":           "model.Geometry",
		"point":              "model.Geometry",
		"linestring":         "model.Geometry",
		"polygon":            "model.Geometry",
		"multipoint":         "model.Geometry",
		"multilinestring":    "model.Geometry",
		"multipolygon":       "model.Geometry",
		"geometrycollection": "model.Geometry",
	}

	goType := "string"
	if t, ok := typeMap[mysqlType]; ok {
		goType = t
	}

	// 处理nullable类型
	if nullable && goType != "[]byte" && goType != "string" {
		switch goType {
		case "int8", "int16", "int32", "int64":
			return "sql.NullInt64"
		case "float32", "float64":
			return "sql.NullFloat64"
		case "bool":
			return "sql.NullBool"
		case "time.Time":
			return "sql.NullTime"
		}
	}

	return goType
}

// buildGormTag 构建GORM标签
func buildGormTag(col ColumnInfo) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("column:%s", col.Name))

	if col.IsPrimary {
		parts = append(parts, "primaryKey")
		if strings.Contains(col.Extra, "auto_increment") {
			parts = append(parts, "autoIncrement")
		}
	}

	tag := fmt.Sprintf("`gorm:\"%s\" json:\"%s\"`", strings.Join(parts, ";"), toSnakeCase(col.Name))
	return tag
}

// toSnakeCase 转换为蛇形命名
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// SqlTypeToBasicType 将 sql.NullXxx 类型转换为基本类型
func SqlTypeToBasicType(goType string) string {
	switch goType {
	case "sql.NullString":
		return "string"
	case "sql.NullInt64":
		return "int64"
	case "sql.NullInt32":
		return "int32"
	case "sql.NullFloat64":
		return "float64"
	case "sql.NullBool":
		return "bool"
	case "sql.NullTime":
		return "time.Time"
	case "model.Geometry":
		// Geometry is converted to JSON string in API responses
		return "string"
	default:
		return goType
	}
}
