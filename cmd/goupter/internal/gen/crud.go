package gen

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

// CRUDConfig CRUD生成配置
type CRUDConfig struct {
	ModelConfig        // 继承 Model 配置
	ServiceName string // 目标服务名（如 user-service, order-service）
	TrimPrefix  string // 过滤表名前缀（如 busi_）
	HandlerDir  string // Handler 输出目录
	TypesDir    string // Types 输出目录（request/response 结构体）
	RoutesFile  string // routes.go 文件路径
	ModuleName  string // Go 模块名
}

// CRUDGenerator CRUD生成器
type CRUDGenerator struct {
	config     CRUDConfig
	modelGen   *ModelGenerator
	tableInfos []*TableInfo
}

// NewCRUDGenerator 创建CRUD生成器
func NewCRUDGenerator(config CRUDConfig) (*CRUDGenerator, error) {
	modelGen, err := NewModelGenerator(config.ModelConfig)
	if err != nil {
		return nil, err
	}

	return &CRUDGenerator{
		config:   config,
		modelGen: modelGen,
	}, nil
}

// Close 关闭
func (g *CRUDGenerator) Close() error {
	return g.modelGen.Close()
}

// Generate 生成 CRUD 代码
func (g *CRUDGenerator) Generate() error {
	// 1. 生成 Model
	fmt.Println("📦 生成 Model...")
	if err := g.modelGen.Generate(); err != nil {
		return fmt.Errorf("生成 Model 失败: %w", err)
	}

	// 2. 获取表信息
	tables, err := g.modelGen.getTables()
	if err != nil {
		return err
	}

	for _, tableName := range tables {
		info, err := g.modelGen.getTableInfo(tableName)
		if err != nil {
			return err
		}
		g.tableInfos = append(g.tableInfos, info)
	}

	// 3. 生成 Types（request/response 结构体）
	fmt.Println("\n📦 生成 Types...")
	if err := os.MkdirAll(g.config.TypesDir, 0755); err != nil {
		return fmt.Errorf("创建 Types 目录失败: %w", err)
	}

	for _, info := range g.tableInfos {
		if err := g.generateTypes(info); err != nil {
			if err == ErrFileExists {
				fmt.Printf("  ⏭ 跳过 %s_types.go（已存在）\n", info.Name)
				continue
			}
			return fmt.Errorf("生成 %s Types 失败: %w", info.Name, err)
		}
		fmt.Printf("  ✓ 生成 %s_types.go\n", info.Name)
	}

	// 4. 生成 Handler
	fmt.Println("\n📦 生成 Handler...")
	if err := os.MkdirAll(g.config.HandlerDir, 0755); err != nil {
		return fmt.Errorf("创建 Handler 目录失败: %w", err)
	}

	for _, info := range g.tableInfos {
		if err := g.generateHandler(info); err != nil {
			if err == ErrFileExists {
				fmt.Printf("  ⏭ 跳过 %s_handler.go（已存在）\n", info.Name)
				continue
			}
			return fmt.Errorf("生成 %s Handler 失败: %w", info.Name, err)
		}
		fmt.Printf("  ✓ 生成 %s_handler.go\n", info.Name)
	}

	// 5. 生成路由注册代码
	fmt.Println("\n📦 生成路由注册...")
	if err := g.generateRoutes(); err != nil {
		return fmt.Errorf("生成路由失败: %w", err)
	}
	fmt.Printf("  ✓ 生成 %s\n", g.config.RoutesFile)

	return nil
}

// TypeField 用于 types 模板的字段信息
type TypeField struct {
	FieldName string
	GoType    string // 基本类型
	Name      string // 原始列名
	JsonName  string // 驼峰格式 JSON 名
	Comment   string
}

// generateTypes 生成 request/response 结构体文件
func (g *CRUDGenerator) generateTypes(info *TableInfo) error {
	// 转换字段为基本类型
	var allFields []TypeField
	var createFields []TypeField
	var updateFields []TypeField
	needTime := false

	for _, col := range info.Columns {
		basicType := SqlTypeToBasicType(col.GoType)
		if basicType == "time.Time" {
			needTime = true
		}

		field := TypeField{
			FieldName: col.FieldName,
			GoType:    basicType,
			Name:      col.Name,
			JsonName:  toCamelCase(col.Name, false), // 小驼峰
			Comment:   col.Comment,
		}
		allFields = append(allFields, field)

		// 跳过主键、created_at、updated_at 等自动字段
		if col.IsPrimary || col.Name == "created_at" || col.Name == "updated_at" || col.Name == "deleted_at" {
			continue
		}
		createFields = append(createFields, field)
		updateFields = append(updateFields, field)
	}

	// 计算 import 路径
	modelImport := path.Join(g.config.ModuleName, "model")
	utilImport := path.Join(g.config.ModuleName, "util")

	data := struct {
		*TableInfo
		AllFields    []TypeField
		CreateFields []TypeField
		UpdateFields []TypeField
		ModelImport  string
		UtilImport   string
		NeedTime     bool
	}{
		TableInfo:    info,
		AllFields:    allFields,
		CreateFields: createFields,
		UpdateFields: updateFields,
		ModelImport:  modelImport,
		UtilImport:   utilImport,
		NeedTime:     needTime,
	}

	tmpl := `package types

import (
{{- if .NeedTime}}
	"time"
{{- end}}

	"{{.ModelImport}}"
	"{{.UtilImport}}"
)

// Create{{.StructName}}Request 创建{{.Comment}}请求
type Create{{.StructName}}Request struct {
{{- range .CreateFields}}
	{{.FieldName}} {{.GoType}} ` + "`" + `json:"{{.JsonName}}"` + "`" + `{{if .Comment}} // {{.Comment}}{{end}}
{{- end}}
}

// ToModel 转换为 Model
func (r *Create{{.StructName}}Request) ToModel() *model.{{.StructName}} {
	m := &model.{{.StructName}}{}
	util.Copy(m, r)
	return m
}

// Update{{.StructName}}Request 更新{{.Comment}}请求
type Update{{.StructName}}Request struct {
{{- range .UpdateFields}}
	{{.FieldName}} *{{.GoType}} ` + "`" + `json:"{{.JsonName}},omitempty"` + "`" + `{{if .Comment}} // {{.Comment}}{{end}}
{{- end}}
}

// ToMap 转换为更新 map（仅非 nil 字段）
func (r *Update{{.StructName}}Request) ToMap() map[string]any {
	m := make(map[string]any)
{{- range .UpdateFields}}
	if r.{{.FieldName}} != nil {
		m["{{.Name}}"] = *r.{{.FieldName}}
	}
{{- end}}
	return m
}

// {{.StructName}}Response {{.Comment}}响应
type {{.StructName}}Response struct {
{{- range .AllFields}}
	{{.FieldName}} {{.GoType}} ` + "`" + `json:"{{.JsonName}}"` + "`" + `{{if .Comment}} // {{.Comment}}{{end}}
{{- end}}
}

// {{.StructName}}ResponseFromModel 从 Model 转换为 Response
func {{.StructName}}ResponseFromModel(m *model.{{.StructName}}) *{{.StructName}}Response {
	if m == nil {
		return nil
	}
	resp := &{{.StructName}}Response{}
	util.Copy(resp, m)
	return resp
}

// List{{.StructName}}Request 列表{{.Comment}}请求
type List{{.StructName}}Request struct {
	Page     int ` + "`" + `form:"page" json:"page"` + "`" + `
	PageSize int ` + "`" + `form:"pageSize" json:"pageSize"` + "`" + `
}

// List{{.StructName}}Response 列表{{.Comment}}响应
type List{{.StructName}}Response struct {
	List     []{{.StructName}}Response ` + "`" + `json:"list"` + "`" + `
	Total    int64                     ` + "`" + `json:"total"` + "`" + `
	Page     int                       ` + "`" + `json:"page"` + "`" + `
	PageSize int                       ` + "`" + `json:"pageSize"` + "`" + `
}
`

	t, err := template.New("types").Parse(tmpl)
	if err != nil {
		return err
	}

	filename := filepath.Join(g.config.TypesDir, info.Name+"_types.go")

	// Skip if file already exists
	if _, err := os.Stat(filename); err == nil {
		return ErrFileExists
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return t.Execute(f, data)
}

// generateHandler 生成 Handler 文件
func (g *CRUDGenerator) generateHandler(info *TableInfo) error {
	// 计算 types 包的 import 路径
	typesImport := path.Join(g.config.ModuleName, "types")
	if g.config.ServiceName != "" {
		typesImport = path.Join(g.config.ModuleName, "cmd", g.config.ServiceName, "types")
	}

	data := struct {
		*TableInfo
		ModuleName    string
		ModelPkg      string
		TypesImport   string
		LowerName     string
		PrimaryKey    string
		PrimaryGoType string
	}{
		TableInfo:     info,
		ModuleName:    g.config.ModuleName,
		ModelPkg:      g.config.PackageName,
		TypesImport:   typesImport,
		LowerName:     strings.ToLower(info.StructName[:1]) + info.StructName[1:],
		PrimaryKey:    info.PrimaryKey,
		PrimaryGoType: g.getPrimaryKeyType(info),
	}

	tmpl := `package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"{{.ModuleName}}/model"
	"{{.TypesImport}}"
	"github.com/goupter/goupter/pkg/errors"
	"github.com/goupter/goupter/pkg/response"
	"gorm.io/gorm"
)

type {{.StructName}}Handler struct {
	model *model.{{.StructName}}Model
}

func New{{.StructName}}Handler(db *gorm.DB) *{{.StructName}}Handler {
	return &{{.StructName}}Handler{
		model: model.New{{.StructName}}Model(db),
	}
}

// Create 创建{{.Comment}}
func (h *{{.StructName}}Handler) Create(c *gin.Context) {
	var req types.Create{{.StructName}}Request
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.New(errors.CodeBadRequest, err.Error()))
		return
	}

	data := req.ToModel()
	if err := h.model.Insert(c.Request.Context(), nil, data); err != nil {
		response.Error(c, errors.Wrap(err, errors.CodeInternalError, "创建失败"))
		return
	}
	response.Success(c, types.{{.StructName}}ResponseFromModel(data))
}

// Get 获取{{.Comment}}
func (h *{{.StructName}}Handler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errors.New(errors.CodeBadRequest, "invalid id"))
		return
	}
	result, err := h.model.FindOne(c.Request.Context(), map[string]any{"{{.PrimaryKey}}": id})
	if err != nil {
		response.Error(c, errors.Wrap(err, errors.CodeNotFound, "记录不存在"))
		return
	}
	response.Success(c, types.{{.StructName}}ResponseFromModel(result))
}

// List 列表{{.Comment}}
func (h *{{.StructName}}Handler) List(c *gin.Context) {
	var req types.List{{.StructName}}Request
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, errors.New(errors.CodeBadRequest, err.Error()))
		return
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	list, total, err := h.model.FindPage(c.Request.Context(), req.Page, req.PageSize, "{{.PrimaryKey}} desc", "")
	if err != nil {
		response.Error(c, err)
		return
	}

	resp := types.List{{.StructName}}Response{
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	for _, item := range list {
		resp.List = append(resp.List, *types.{{.StructName}}ResponseFromModel(item))
	}
	response.Success(c, resp)
}

// Update 更新{{.Comment}}
func (h *{{.StructName}}Handler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errors.New(errors.CodeBadRequest, "invalid id"))
		return
	}
	var req types.Update{{.StructName}}Request
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.New(errors.CodeBadRequest, err.Error()))
		return
	}

	updates := req.ToMap()
	if err := h.model.UpdateColumns(c.Request.Context(), nil, map[string]any{"{{.PrimaryKey}}": id}, updates); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"updated": true})
}

// Delete 删除{{.Comment}}
func (h *{{.StructName}}Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errors.New(errors.CodeBadRequest, "invalid id"))
		return
	}
	if err := h.model.Delete(c.Request.Context(), nil, map[string]any{"{{.PrimaryKey}}": id}); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
`

	t, err := template.New("handler").Parse(tmpl)
	if err != nil {
		return err
	}

	filename := filepath.Join(g.config.HandlerDir, info.Name+"_handler.go")

	// Skip if file already exists
	if _, err := os.Stat(filename); err == nil {
		return ErrFileExists
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return t.Execute(f, data)
}

// generateRoutes 生成路由注册文件（追加模式）
func (g *CRUDGenerator) generateRoutes() error {
	routesPkg := "handler"
	useHandlerPkg := false
	handlerImport := ""
	if strings.TrimSpace(g.config.ServiceName) != "" {
		routesPkg = "main"
		useHandlerPkg = true
		handlerImport = path.Join(g.config.ModuleName, "cmd", g.config.ServiceName, "handler")
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(g.config.RoutesFile), 0755); err != nil {
		return fmt.Errorf("创建路由目录失败: %w", err)
	}

	// 检查文件是否存在
	fileExists := false
	existingFuncs := make(map[string]bool)
	if _, err := os.Stat(g.config.RoutesFile); err == nil {
		fileExists = true
		existingFuncs = g.parseExistingRouteFuncs()
	}

	// 过滤掉已存在的表
	var newTables []*TableInfo
	for _, t := range g.tableInfos {
		funcName := "Register" + t.StructName + "Routes"
		if !existingFuncs[funcName] {
			newTables = append(newTables, t)
		} else {
			fmt.Printf("  ⏭ 跳过 %s（已存在）\n", funcName)
		}
	}

	if len(newTables) == 0 {
		fmt.Println("  ℹ 无新路由需要生成")
		return nil
	}

	if !fileExists {
		// 创建新文件，写入头部
		if err := g.writeRoutesHeader(routesPkg, handlerImport); err != nil {
			return err
		}
	} else {
		// 确保必要的 import 存在
		if err := g.ensureImports(handlerImport); err != nil {
			return err
		}
	}

	// 追加路由函数
	return g.appendRouteFuncs(newTables, useHandlerPkg)
}

// parseExistingRouteFuncs 解析已存在的路由函数名
func (g *CRUDGenerator) parseExistingRouteFuncs() map[string]bool {
	funcs := make(map[string]bool)
	f, err := os.Open(g.config.RoutesFile)
	if err != nil {
		return funcs
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "func Register") && strings.Contains(line, "Routes(") {
			// 提取函数名
			parts := strings.Split(line, "(")
			if len(parts) > 0 {
				funcName := strings.TrimPrefix(parts[0], "func ")
				funcs[funcName] = true
			}
		}
	}
	return funcs
}

// ensureImports 确保必要的 import 存在
func (g *CRUDGenerator) ensureImports(handlerImport string) error {
	content, err := os.ReadFile(g.config.RoutesFile)
	if err != nil {
		return err
	}

	fileContent := string(content)
	modified := false

	// 检查必要的 import
	requiredImports := []string{
		`"github.com/gin-gonic/gin"`,
		`"gorm.io/gorm"`,
	}
	if handlerImport != "" {
		requiredImports = append(requiredImports, `"`+handlerImport+`"`)
	}

	for _, imp := range requiredImports {
		if !strings.Contains(fileContent, imp) {
			// 找到 import 块并添加
			fileContent, modified = g.addImport(fileContent, imp), true
		}
	}

	if modified {
		return os.WriteFile(g.config.RoutesFile, []byte(fileContent), 0644)
	}
	return nil
}

// addImport 向文件添加 import
func (g *CRUDGenerator) addImport(content, newImport string) string {
	// 查找 import ( 块
	importStart := strings.Index(content, "import (")
	if importStart == -1 {
		// 没有 import 块，在 package 行后添加
		pkgEnd := strings.Index(content, "\n")
		if pkgEnd == -1 {
			return content
		}
		return content[:pkgEnd+1] + "\nimport (\n\t" + newImport + "\n)\n" + content[pkgEnd+1:]
	}

	// 找到 import 块的结束位置
	importEnd := strings.Index(content[importStart:], ")")
	if importEnd == -1 {
		return content
	}
	importEnd += importStart

	// 在 ) 前插入新 import
	return content[:importEnd] + "\t" + newImport + "\n" + content[importEnd:]
}

// writeRoutesHeader 写入路由文件头部
func (g *CRUDGenerator) writeRoutesHeader(pkg, handlerImport string) error {
	data := struct {
		Package       string
		HandlerImport string
	}{
		Package:       pkg,
		HandlerImport: handlerImport,
	}

	tmpl := `package {{.Package}}

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
{{- if .HandlerImport}}

	"{{.HandlerImport}}"
{{- end}}
)
`
	t, err := template.New("header").Parse(tmpl)
	if err != nil {
		return err
	}

	f, err := os.Create(g.config.RoutesFile)
	if err != nil {
		return err
	}
	defer f.Close()

	return t.Execute(f, data)
}

// appendRouteFuncs 追加路由函数
func (g *CRUDGenerator) appendRouteFuncs(tables []*TableInfo, useHandlerPkg bool) error {
	f, err := os.OpenFile(g.config.RoutesFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	tmpl := `
// Register{{.StructName}}Routes 注册 {{.Comment}} 路由
func Register{{.StructName}}Routes(r *gin.RouterGroup, db *gorm.DB) {
	h := {{if .UseHandlerPkg}}handler.{{end}}New{{.StructName}}Handler(db)
	g := r.Group("/{{.Name}}s")
	{
		g.POST("", h.Create)
		g.GET("/:id", h.Get)
		g.GET("", h.List)
		g.PUT("/:id", h.Update)
		g.DELETE("/:id", h.Delete)
	}
}
`
	t, err := template.New("route").Parse(tmpl)
	if err != nil {
		return err
	}

	for _, table := range tables {
		data := struct {
			*TableInfo
			UseHandlerPkg bool
		}{
			TableInfo:     table,
			UseHandlerPkg: useHandlerPkg,
		}
		if err := t.Execute(f, data); err != nil {
			return err
		}
		fmt.Printf("  ✓ 追加 Register%sRoutes\n", table.StructName)
	}

	return nil
}

func (g *CRUDGenerator) getPrimaryKeyType(info *TableInfo) string {
	for _, col := range info.Columns {
		if col.IsPrimary {
			return col.GoType
		}
	}
	return "int64"
}
