package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new [service-name]",
	Short: "创建一个新的微服务",
	Long: `在当前目录创建一个新的微服务。

支持在同一项目下创建多个服务，共享 go.mod、Makefile 等文件。

示例:
  goupter new user              # 创建 user 服务
  goupter new order             # 创建 order 服务
  goupter new user --module github.com/myorg/myproject  # 首次创建时指定模块名`,
	Args: cobra.ExactArgs(1),
	Run:  runNew,
}

var moduleName string

func init() {
	rootCmd.AddCommand(newCmd)
	newCmd.Flags().StringVarP(&moduleName, "module", "m", "", "Go模块名称（仅首次创建时需要）")
}

func runNew(cmd *cobra.Command, args []string) {
	serviceName := args[0]

	// 检查是否已有 go.mod（判断是否为已有项目）
	isNewProject := !fileExists("go.mod")

	if isNewProject {
		if moduleName == "" {
			// 使用当前目录名作为模块名
			cwd, _ := os.Getwd()
			moduleName = filepath.Base(cwd)
		}
		createSharedFiles(moduleName)
	} else {
		// 从现有 go.mod 读取模块名
		moduleName = readGoModuleName()
		if moduleName == "" {
			moduleName = "myproject"
		}
	}

	// 创建服务目录结构
	createServiceDirs(serviceName)
	createServiceFiles(serviceName, moduleName)

	fmt.Printf("✅ 服务 %s 创建成功!\n\n", serviceName)
	if isNewProject {
		fmt.Println("首次创建，请执行:")
		fmt.Println("  go mod tidy")
	}
	fmt.Printf("启动服务:\n  go run ./cmd/%s\n", serviceName)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func createSharedFiles(module string) {
	writeFileIfNotExists("go.mod", generateGoMod(module))
	writeFileIfNotExists(".gitignore", generateGitignore())
	writeFileIfNotExists("Makefile", generateMakefile())
	// 创建共享 util 目录
	if err := os.MkdirAll("util", 0755); err != nil {
		exitWithError("创建 util 目录失败", err)
	}
	writeFileIfNotExists("util/copy.go", generateCopyUtil())
	writeFileIfNotExists("util/geometry.go", generateGeometryUtil())
}

func writeFileIfNotExists(path string, content string) {
	if fileExists(path) {
		return
	}
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			exitWithError(fmt.Sprintf("创建目录 %s 失败", dir), err)
		}
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		exitWithError(fmt.Sprintf("写入文件 %s 失败", path), err)
	}
}

func createServiceDirs(name string) {
	dirs := []string{
		fmt.Sprintf("cmd/%s/config", name),
		fmt.Sprintf("cmd/%s/handler", name),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			exitWithError(fmt.Sprintf("创建目录 %s 失败", dir), err)
		}
	}
}

func createServiceFiles(name, module string) {
	files := map[string]string{
		fmt.Sprintf("cmd/%s/main.go", name):            generateMainGo(name, module),
		fmt.Sprintf("cmd/%s/routes.go", name):          generateRoutesGo(name, module),
		fmt.Sprintf("cmd/%s/handler/health.go", name):  generateHealthHandler(name),
		fmt.Sprintf("cmd/%s/config/config.yaml", name): generateConfigYaml(name),
	}

	for path, content := range files {
		if fileExists(path) {
			fmt.Printf("  ⚠️  跳过已存在: %s\n", path)
			continue
		}
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			exitWithError(fmt.Sprintf("创建目录 %s 失败", dir), err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			exitWithError(fmt.Sprintf("创建文件 %s 失败", path), err)
		}
	}
}

func generateGoMod(module string) string {
	return fmt.Sprintf(`module %s

go 1.21

require (
	github.com/goupter/goupter v0.0.1
)
`, module)
}

func generateMainGo(name, module string) string {
	return fmt.Sprintf(`package main

import (
	"log"

	"github.com/goupter/goupter/pkg/app"
	"github.com/goupter/goupter/pkg/cache"
	"github.com/goupter/goupter/pkg/config"
	"github.com/goupter/goupter/pkg/database"
	pkglog "github.com/goupter/goupter/pkg/log"
	"github.com/goupter/goupter/pkg/server"
	"github.com/goupter/goupter/pkg/server/middleware"
)

func main() {
	// 1. Load configuration
	// Search paths:
	// - "./" : 部署时，配置文件与二进制文件同目录
	// - "./config" : 开发时，从 cmd/%s 目录运行
	// - "./cmd/%s/config" : 开发时，从项目根目录运行
	cfg, err := config.Load(
		config.WithConfigFile("config"),
		config.WithConfigPaths("./", "./config", "./cmd/%s/config"),
	)
	if err != nil {
		log.Fatalf("Failed to load config: %%v", err)
	}

	// 2. Initialize logger
	logger, err := pkglog.NewZapLogger(&pkglog.LoggerConfig{
		Level:  pkglog.ParseLevel(cfg.Log.Level),
		Format: cfg.Log.Format,
		Output: cfg.Log.Output,
	})
	if err != nil {
		log.Fatalf("Failed to create logger: %%v", err)
	}
	pkglog.SetDefault(logger)

	// 3. Initialize database
	mysqlDB, err := database.NewMySQL(
		database.WithConfig(&cfg.Database),
		database.WithLogger(logger),
	)
	if err != nil {
		log.Fatalf("Failed to connect database: %%v", err)
	}
	db := mysqlDB.DB()

	// 4. Initialize cache (Redis)
	redisCache, err := cache.NewRedisCache(
		cache.WithRedisConfig(&cfg.Redis),
	)
	if err != nil {
		log.Fatalf("Failed to connect Redis: %%v", err)
	}

	// 5. Create HTTP server with middleware
	httpServer := server.NewHTTPServer(
		server.WithHTTPConfig(&cfg.Server.HTTP),
		server.WithHTTPLogger(logger),
		server.WithMiddleware(
			middleware.Recovery(logger),
			middleware.Logger(logger),
			middleware.Security(),
			middleware.CORS(),
		),
	)

	// 6. Create health manager
	healthManager := app.NewHealthManager(app.DefaultHealthConfig())
	healthManager.RegisterReadiness(app.NewPingChecker("cache", redisCache.Ping))

	// 7. Create application
	application := app.New(
		app.WithName(cfg.App.Name),
		app.WithConfig(cfg),
		app.WithLogger(logger),
		app.WithHTTPServer(httpServer),
		app.WithDatabase(db),
		app.WithCache(redisCache),
		app.WithHealthManager(healthManager),
		app.BeforeStart(func() error {
			logger.Info("Application starting...")
			return nil
		}),
		app.AfterStart(func() error {
			logger.Info("Application started",
				pkglog.String("name", cfg.App.Name),
				pkglog.String("addr", cfg.HTTPAddr()),
			)
			return nil
		}),
		app.BeforeStop(func() error {
			logger.Info("Application stopping...")
			return nil
		}),
	)

	// 8. Register routes
	registerRoutes(application, db, redisCache)

	// 9. Run application
	if err := application.Run(); err != nil {
		log.Fatalf("Application error: %%v", err)
	}
}
`, name, name, name)
}

func generateRoutesGo(name, module string) string {
	return fmt.Sprintf(`package main

import (
	"github.com/gin-gonic/gin"
	"%s/cmd/%s/handler"
	"github.com/goupter/goupter/pkg/app"
	"github.com/goupter/goupter/pkg/cache"
	"github.com/goupter/goupter/pkg/response"
	"gorm.io/gorm"
)

func registerRoutes(a *app.App, db *gorm.DB, c cache.Cache) {
	s := a.HTTPServer()

	// Root endpoint
	s.GET("/", func(c *gin.Context) {
		cfg := a.Config()
		response.Success(c, gin.H{
			"service": "%s",
			"version": cfg.App.Version,
		})
	})

	// API v1 routes
	api := s.Group("/api/v1")
	{
		api.GET("/health", handler.Health)
	}

	// TODO: Add your routes here
	// Example with authentication:
	// tokenManager := token.NewManager(token.NewConfigFromMap(a.Config().Auth.Config), c)
	// auth := api.Group("", tokenManager.JWTAuth())
	// {
	//     auth.GET("/me", token.WrapUser(handler.Me))
	// }
}
`, module, name, name)
}

func generateHealthHandler(name string) string {
	return fmt.Sprintf(`package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/goupter/goupter/pkg/response"
)

func Health(c *gin.Context) {
	response.Success(c, gin.H{
		"service": "%s",
		"status":  "ok",
	})
}
`, name)
}

func generateConfigYaml(name string) string {
	return fmt.Sprintf(`app:
  name: %s
  version: 1.0.0
  env: development

server:
  http:
    host: 0.0.0.0
    port: 8080

log:
  level: debug
  format: console
  output: stdout

database:
  driver: mysql
  host: localhost
  port: 3306
  database: %s
  username: root
  password: ""
  max_idle_conns: 10
  max_open_conns: 100
  conn_max_lifetime: 3600

redis:
  host: localhost
  port: 6379
  password: ""
  db: 0

auth:
  type: jwt
  config:
    secret_key: "%s-secret-key-at-least-32-chars"
    issuer: "%s"
    access_token_ttl: "2h"
`, name, name, name, name)
}

func generateGitignore() string {
	return `# Binaries
*.exe
*.dll
*.so
*.dylib
bin/

# Test
*.test
*.out
coverage.html

# Vendor
vendor/

# IDE
.idea/
.vscode/
*.swp

# OS
.DS_Store

# Config
*.local.yaml
.env
`
}

func generateMakefile() string {
	return `.PHONY: build run test clean

BIN_DIR=bin

# 构建所有服务
build:
	@for dir in cmd/*; do \
		if [ -d "$$dir" ]; then \
			name=$$(basename $$dir); \
			echo "Building $$name..."; \
			go build -o $(BIN_DIR)/$$name ./$$dir; \
		fi \
	done

# 运行指定服务: make run SVC=user
run:
	go run ./cmd/$(SVC)

test:
	go test -v -race -cover ./...

clean:
	rm -rf $(BIN_DIR)

tidy:
	go mod tidy
`
}

func generateCopyUtil() string {
	return `package util

import (
	"database/sql"
	"reflect"
	"sync"
	"time"

	"github.com/goupter/goupter/pkg/model"
)

var fieldCache sync.Map // map[reflect.Type]map[string]int

// Copy 复制同名字段从 src 到 dst（dst 必须是指针）
func Copy(dst, src any) {
	dstVal := reflect.ValueOf(dst)
	if dstVal.Kind() != reflect.Ptr || dstVal.IsNil() {
		return
	}
	dstVal = dstVal.Elem()
	srcVal := reflect.ValueOf(src)
	if srcVal.Kind() == reflect.Ptr {
		if srcVal.IsNil() {
			return
		}
		srcVal = srcVal.Elem()
	}

	dstFields := getFieldMap(dstVal.Type())
	srcFields := getFieldMap(srcVal.Type())

	for name, dstIdx := range dstFields {
		srcIdx, ok := srcFields[name]
		if !ok {
			continue
		}
		dstField := dstVal.Field(dstIdx)
		srcField := srcVal.Field(srcIdx)
		if !dstField.CanSet() {
			continue
		}
		copyField(dstField, srcField)
	}
}

func getFieldMap(t reflect.Type) map[string]int {
	if cached, ok := fieldCache.Load(t); ok {
		return cached.(map[string]int)
	}
	m := make(map[string]int)
	for i := 0; i < t.NumField(); i++ {
		m[t.Field(i).Name] = i
	}
	fieldCache.Store(t, m)
	return m
}

func copyField(dst, src reflect.Value) {
	srcType := src.Type()
	dstType := dst.Type()

	// 类型相同直接赋值
	if srcType == dstType {
		dst.Set(src)
		return
	}

	// model.Geometry -> string (WKT to JSON coordinates)
	if srcType.PkgPath() == "github.com/goupter/goupter/pkg/model" && srcType.Name() == "Geometry" {
		if dstType.Kind() == reflect.String {
			wkt := string(src.Interface().(model.Geometry))
			jsonStr := WKTToJSON(wkt)
			dst.SetString(jsonStr)
		}
		return
	}

	// string -> model.Geometry
	if dstType.PkgPath() == "github.com/goupter/goupter/pkg/model" && dstType.Name() == "Geometry" {
		if srcType.Kind() == reflect.String {
			dst.Set(reflect.ValueOf(model.Geometry(src.String())))
		}
		return
	}

	// sql.Null* -> 基本类型
	if srcType.PkgPath() == "database/sql" {
		switch srcType.Name() {
		case "NullString":
			if dstType.Kind() == reflect.String {
				dst.SetString(src.Interface().(sql.NullString).String)
			}
		case "NullInt64":
			if dstType.Kind() == reflect.Int64 {
				dst.SetInt(src.Interface().(sql.NullInt64).Int64)
			}
		case "NullFloat64":
			if dstType.Kind() == reflect.Float64 {
				dst.SetFloat(src.Interface().(sql.NullFloat64).Float64)
			}
		case "NullBool":
			if dstType.Kind() == reflect.Bool {
				dst.SetBool(src.Interface().(sql.NullBool).Bool)
			}
		case "NullTime":
			if dstType == reflect.TypeOf(time.Time{}) {
				dst.Set(reflect.ValueOf(src.Interface().(sql.NullTime).Time))
			}
		}
		return
	}

	// 基本类型 -> sql.Null*
	if dstType.PkgPath() == "database/sql" {
		switch dstType.Name() {
		case "NullString":
			if srcType.Kind() == reflect.String {
				dst.Set(reflect.ValueOf(sql.NullString{String: src.String(), Valid: src.String() != ""}))
			}
		case "NullInt64":
			if srcType.Kind() == reflect.Int64 {
				dst.Set(reflect.ValueOf(sql.NullInt64{Int64: src.Int(), Valid: true}))
			}
		case "NullFloat64":
			if srcType.Kind() == reflect.Float64 {
				dst.Set(reflect.ValueOf(sql.NullFloat64{Float64: src.Float(), Valid: true}))
			}
		case "NullBool":
			if srcType.Kind() == reflect.Bool {
				dst.Set(reflect.ValueOf(sql.NullBool{Bool: src.Bool(), Valid: true}))
			}
		case "NullTime":
			if srcType == reflect.TypeOf(time.Time{}) {
				t := src.Interface().(time.Time)
				dst.Set(reflect.ValueOf(sql.NullTime{Time: t, Valid: !t.IsZero()}))
			}
		}
		return
	}

	// 可转换类型
	if srcType.ConvertibleTo(dstType) {
		dst.Set(src.Convert(dstType))
	}
}
`
}

func generateGeometryUtil() string {
	return "package util\n\nimport (\n\t\"encoding/json\"\n\t\"regexp\"\n\t\"strconv\"\n\t\"strings\"\n)\n\n// Coord represents a coordinate point with longitude and latitude\ntype Coord struct {\n\tLng float64 `json:\"lng\"`\n\tLat float64 `json:\"lat\"`\n}\n\n// WKTToJSON converts WKT (Well-Known Text) geometry to JSON coordinate array\n// POINT, LINESTRING -> 1D array: [{\"lng\":x,\"lat\":y},...]\n// MULTIPOINT, MULTILINESTRING, POLYGON, MULTIPOLYGON -> 2D array: [[{\"lng\":x,\"lat\":y},...],...]" +
		"\nfunc WKTToJSON(wkt string) string {\n\tif wkt == \"\" {\n\t\treturn \"\"\n\t}\n\n\twkt = strings.TrimSpace(wkt)\n\tupperWKT := strings.ToUpper(wkt)\n\n\tswitch {\n\tcase strings.HasPrefix(upperWKT, \"POINT\"):\n\t\treturn pointToJSON(wkt)\n\tcase strings.HasPrefix(upperWKT, \"LINESTRING\"):\n\t\treturn lineStringToJSON(wkt)\n\tcase strings.HasPrefix(upperWKT, \"POLYGON\"):\n\t\treturn polygonToJSON(wkt)\n\tcase strings.HasPrefix(upperWKT, \"MULTIPOINT\"):\n\t\treturn multiPointToJSON(wkt)\n\tcase strings.HasPrefix(upperWKT, \"MULTILINESTRING\"):\n\t\treturn multiLineStringToJSON(wkt)\n\tcase strings.HasPrefix(upperWKT, \"MULTIPOLYGON\"):\n\t\treturn multiPolygonToJSON(wkt)\n\tdefault:\n\t\treturn wkt\n\t}\n}\n\n// parseCoordPair parses \"lng lat\" string to Coord\nfunc parseCoordPair(s string) *Coord {\n\ts = strings.TrimSpace(s)\n\tparts := regexp.MustCompile(`\\s+`).Split(s, 2)\n\tif len(parts) != 2 {\n\t\treturn nil\n\t}\n\tlng, err1 := strconv.ParseFloat(parts[0], 64)\n\tlat, err2 := strconv.ParseFloat(parts[1], 64)\n\tif err1 != nil || err2 != nil {\n\t\treturn nil\n\t}\n\treturn &Coord{Lng: lng, Lat: lat}\n}\n\n// parseCoordList parses \"lng1 lat1, lng2 lat2, ...\" to []Coord\nfunc parseCoordList(s string) []Coord {\n\ts = strings.TrimSpace(s)\n\tif s == \"\" {\n\t\treturn nil\n\t}\n\tpairs := strings.Split(s, \",\")\n\tcoords := make([]Coord, 0, len(pairs))\n\tfor _, pair := range pairs {\n\t\tc := parseCoordPair(pair)\n\t\tif c != nil {\n\t\t\tcoords = append(coords, *c)\n\t\t}\n\t}\n\treturn coords\n}\n\n// extractContent extracts content between first ( and last )\nfunc extractContent(wkt, prefix string) string {\n\tidx := strings.Index(strings.ToUpper(wkt), prefix)\n\tif idx == -1 {\n\t\treturn \"\"\n\t}\n\trest := wkt[idx+len(prefix):]\n\tstart := strings.Index(rest, \"(\")\n\tif start == -1 {\n\t\treturn \"\"\n\t}\n\tend := strings.LastIndex(rest, \")\")\n\tif end == -1 || end <= start {\n\t\treturn \"\"\n\t}\n\treturn rest[start+1 : end]\n}\n\n// pointToJSON: POINT(lng lat) -> [{\"lng\":x,\"lat\":y}]\nfunc pointToJSON(wkt string) string {\n\tcontent := extractContent(wkt, \"POINT\")\n\tc := parseCoordPair(content)\n\tif c == nil {\n\t\treturn \"[]\"\n\t}\n\tb, _ := json.Marshal([]Coord{*c})\n\treturn string(b)\n}\n\n// lineStringToJSON: LINESTRING(lng1 lat1, lng2 lat2) -> [{\"lng\":x,\"lat\":y},...]\nfunc lineStringToJSON(wkt string) string {\n\tcontent := extractContent(wkt, \"LINESTRING\")\n\tcoords := parseCoordList(content)\n\tif len(coords) == 0 {\n\t\treturn \"[]\"\n\t}\n\tb, _ := json.Marshal(coords)\n\treturn string(b)\n}\n\n// polygonToJSON: POLYGON((lng1 lat1, ...)) -> [[{\"lng\":x,\"lat\":y},...]]\nfunc polygonToJSON(wkt string) string {\n\tcontent := extractContent(wkt, \"POLYGON\")\n\trings := parseRings(content)\n\tif len(rings) == 0 {\n\t\treturn \"[]\"\n\t}\n\tb, _ := json.Marshal(rings)\n\treturn string(b)\n}\n\n// multiPointToJSON: MULTIPOINT((lng1 lat1), (lng2 lat2)) -> [[{\"lng\":x,\"lat\":y}],[...]]\nfunc multiPointToJSON(wkt string) string {\n\tcontent := extractContent(wkt, \"MULTIPOINT\")\n\t// MULTIPOINT can be ((x y), (x y)) or (x y, x y)\n\tcontent = strings.ReplaceAll(content, \"(\", \"\")\n\tcontent = strings.ReplaceAll(content, \")\", \"\")\n\tcoords := parseCoordList(content)\n\tif len(coords) == 0 {\n\t\treturn \"[]\"\n\t}\n\t// Return as 2D array for MULTI types\n\tresult := make([][]Coord, len(coords))\n\tfor i, c := range coords {\n\t\tresult[i] = []Coord{c}\n\t}\n\tb, _ := json.Marshal(result)\n\treturn string(b)\n}\n\n// multiLineStringToJSON: MULTILINESTRING((x y, x y), (x y, x y)) -> [[{},{}],[{},{}]]\nfunc multiLineStringToJSON(wkt string) string {\n\tcontent := extractContent(wkt, \"MULTILINESTRING\")\n\trings := parseRings(content)\n\tif len(rings) == 0 {\n\t\treturn \"[]\"\n\t}\n\tb, _ := json.Marshal(rings)\n\treturn string(b)\n}\n\n// multiPolygonToJSON: MULTIPOLYGON(((x y, ...)), ((x y, ...))) -> [[[{},...]],[[{}...]]]\nfunc multiPolygonToJSON(wkt string) string {\n\tcontent := extractContent(wkt, \"MULTIPOLYGON\")\n\tpolygons := parsePolygons(content)\n\tif len(polygons) == 0 {\n\t\treturn \"[]\"\n\t}\n\tb, _ := json.Marshal(polygons)\n\treturn string(b)\n}\n\n// parseRings parses \"(x y, x y), (x y, x y)\" to [][]Coord\nfunc parseRings(content string) [][]Coord {\n\tvar rings [][]Coord\n\tdepth := 0\n\tstart := -1\n\tfor i, c := range content {\n\t\tif c == '(' {\n\t\t\tif depth == 0 {\n\t\t\t\tstart = i + 1\n\t\t\t}\n\t\t\tdepth++\n\t\t} else if c == ')' {\n\t\t\tdepth--\n\t\t\tif depth == 0 && start >= 0 {\n\t\t\t\tringContent := content[start:i]\n\t\t\t\tcoords := parseCoordList(ringContent)\n\t\t\t\tif len(coords) > 0 {\n\t\t\t\t\trings = append(rings, coords)\n\t\t\t\t}\n\t\t\t\tstart = -1\n\t\t\t}\n\t\t}\n\t}\n\treturn rings\n}\n\n// parsePolygons parses \"((...), (...)), ((...), (...))\" to [][][]Coord\nfunc parsePolygons(content string) [][][]Coord {\n\tvar polygons [][][]Coord\n\tdepth := 0\n\tstart := -1\n\tfor i, c := range content {\n\t\tif c == '(' {\n\t\t\tif depth == 0 {\n\t\t\t\tstart = i + 1\n\t\t\t}\n\t\t\tdepth++\n\t\t} else if c == ')' {\n\t\t\tdepth--\n\t\t\tif depth == 0 && start >= 0 {\n\t\t\t\tpolyContent := content[start:i]\n\t\t\t\trings := parseRings(polyContent)\n\t\t\t\tif len(rings) > 0 {\n\t\t\t\t\tpolygons = append(polygons, rings)\n\t\t\t\t}\n\t\t\t\tstart = -1\n\t\t\t}\n\t\t}\n\t}\n\treturn polygons\n}\n"
}
