package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var genHTTPBookCmd = &cobra.Command{
	Use:   "httpbook",
	Short: "生成 HTTP API 调试文件（.http 格式）",
	Long: `扫描服务目录下的 handler 和 routes.go 生成 HTTP API 调试文件。

示例:
  # 生成指定服务的 httpbook
  goupter gen httpbook --service drone

  # 指定输出目录和端口
  goupter gen httpbook --service drone --out-dir ./httpbook --port 8081`,
	Run: runGenHTTPBook,
}

var (
	httpbookService string
	httpbookOutDir  string
	httpbookBaseURL string
	httpbookPort    string
)

func init() {
	genCmd.AddCommand(genHTTPBookCmd)

	genHTTPBookCmd.Flags().StringVar(&httpbookService, "service", "", "服务名 (必需)")
	genHTTPBookCmd.Flags().StringVar(&httpbookOutDir, "out-dir", "./httpbook", "输出目录")
	genHTTPBookCmd.Flags().StringVar(&httpbookBaseURL, "base-url", "http://localhost", "基础 URL")
	genHTTPBookCmd.Flags().StringVar(&httpbookPort, "port", "8080", "服务端口")

	genHTTPBookCmd.MarkFlagRequired("service")
}

// RouteInfo 路由信息
type RouteInfo struct {
	Name       string // 资源名（如 drone_info）
	StructName string // 结构体名（如 DroneInfo）
	Comment    string // 注释
	BasePath   string // 基础路径（如 /drone_infos）
	HasCreate  bool
	HasGet     bool
	HasList    bool
	HasUpdate  bool
	HasDelete  bool
	Fields     []FieldInfo // 字段信息（从 types 解析）
	// 新增：存储完整的端点信息
	Endpoints []EndpointInfo
}

// EndpointInfo 端点信息
type EndpointInfo struct {
	Method   string // GET, POST, PUT, DELETE
	Path     string // 完整路径
	IsDetail bool   // 是否是详情端点（包含 :id）
}

// FieldInfo 字段信息
type FieldInfo struct {
	Name    string
	Type    string
	JsonTag string
}

func runGenHTTPBook(cmd *cobra.Command, args []string) {
	serviceDir := filepath.Join("cmd", httpbookService)
	if _, err := os.Stat(serviceDir); os.IsNotExist(err) {
		exitWithError(fmt.Sprintf("服务目录不存在: %s", serviceDir), nil)
	}

	// 解析 routes.go
	routesFile := filepath.Join(serviceDir, "routes.go")
	routes := parseRoutesFile(routesFile)

	if len(routes) == 0 {
		fmt.Println("⚠️  未找到路由定义")
	}

	// 解析 types 目录获取字段信息
	typesDir := filepath.Join(serviceDir, "types")
	for i := range routes {
		routes[i].Fields = parseTypesFile(typesDir, routes[i].Name)
	}

	// 创建输出目录
	if err := os.MkdirAll(httpbookOutDir, 0755); err != nil {
		exitWithError("创建输出目录失败", err)
	}

	// 生成 httpbook 文件
	filename := filepath.Join(httpbookOutDir, httpbookService+".http")
	content := generateHTTPBookFromRoutes(httpbookService, httpbookBaseURL, httpbookPort, routes)

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		exitWithError("写入文件失败", err)
	}

	fmt.Printf("✅ 生成成功: %s\n", filename)
}

// parseRoutesFile 解析 routes.go 文件
func parseRoutesFile(filename string) []RouteInfo {
	var routes []RouteInfo

	f, err := os.Open(filename)
	if err != nil {
		return routes
	}
	defer f.Close()

	// 读取整个文件内容
	content, err := os.ReadFile(filename)
	if err != nil {
		return routes
	}
	fileContent := string(content)

	// 正则匹配路由注册函数（支持多种格式）
	funcRe := regexp.MustCompile(`func [Rr]egister(\w*)Routes?\(`)

	// 路由组匹配（支持多种变量名和调用方式，包括带额外参数的情况）
	groupRe := regexp.MustCompile(`\b(\w+)\s*:?=\s*(\w+)\.Group\("([^"]*)"`)
	httpServerRe := regexp.MustCompile(`\b(\w+)\s*:?=\s*\w+\.HTTPServer\(\)`)

	// 方法匹配（支持多种变量名）
	methodRe := regexp.MustCompile(`(\w+)\.(POST|GET|PUT|DELETE)\("([^"]*)"`)

	// 注释匹配
	commentRe := regexp.MustCompile(`//\s*[Rr]egister\w*Routes?\s+注册\s*(.+)\s*路由`)

	// 函数调用匹配：RegisterXxxRoutes(api, db) 或 RegisterXxxRoutes(r, db)
	funcCallRe := regexp.MustCompile(`[Rr]egister(\w*)Routes?\((\w+),`)

	// 函数参数匹配：func RegisterXxxRoutes(r *gin.RouterGroup, ...)
	// 使用更宽松的匹配，支持多种空格格式
	funcParamRe := regexp.MustCompile(`func [Rr]egister\w*Routes?\((\w+)\s+\*gin\.RouterGroup`)

	// 第一遍：解析主函数中的路由组定义和函数调用
	var globalGroupPaths = make(map[string]string) // 变量名 -> 累积路径
	var funcCallParams = make(map[string]string)   // 函数名 -> 调用时传入的变量名

	// 匹配主路由注册函数（支持多种命名：registerRoutes, RegisterRoutes, registerRoute 等）
	mainFuncRe := regexp.MustCompile(`func\s+[rR]egister[rR]outes?\s*\(`)

	scanner := bufio.NewScanner(strings.NewReader(fileContent))
	inMainFunc := false
	braceCount := 0

	for scanner.Scan() {
		line := scanner.Text()

		// 检查是否进入主路由注册函数（支持多种命名格式）
		if mainFuncRe.MatchString(line) {
			inMainFunc = true
			braceCount = 0
		}

		if inMainFunc {
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")

			// 检查 HTTPServer 赋值
			if matches := httpServerRe.FindStringSubmatch(line); len(matches) > 1 {
				varName := matches[1]
				globalGroupPaths[varName] = ""
			}

			// 检查路由组定义
			if matches := groupRe.FindStringSubmatch(line); len(matches) > 3 {
				newVar := matches[1]
				parentVar := matches[2]
				groupPath := matches[3]

				parentPath := globalGroupPaths[parentVar]
				fullPath := parentPath + groupPath
				globalGroupPaths[newVar] = fullPath
			}

			// 检查函数调用
			if matches := funcCallRe.FindStringSubmatch(line); len(matches) > 2 {
				funcName := "Register" + matches[1] + "Routes"
				paramVar := matches[2]
				funcCallParams[funcName] = paramVar
			}

			if braceCount == 0 && strings.Contains(line, "}") {
				inMainFunc = false
			}
		}
	}

	// 第二遍：解析所有路由注册函数
	f.Seek(0, 0)
	scanner = bufio.NewScanner(f)
	var lastComment string
	var groupPaths = make(map[string]string)
	var inFunction bool
	var currentFuncName string
	var routeMap = make(map[string]*RouteInfo)

	for scanner.Scan() {
		line := scanner.Text()

		// 检查注释
		if matches := commentRe.FindStringSubmatch(line); len(matches) > 1 {
			lastComment = matches[1]
			continue
		}

		// 检查函数定义
		if matches := funcRe.FindStringSubmatch(line); len(matches) > 1 {
			inFunction = true
			currentFuncName = "Register" + matches[1] + "Routes"
			groupPaths = make(map[string]string)

			// 检查函数参数，将参数名映射到调用时传入的变量的路径
			if paramMatches := funcParamRe.FindStringSubmatch(line); len(paramMatches) > 1 {
				paramName := paramMatches[1]
				// 查找调用时传入的变量名
				if callerVar, ok := funcCallParams[currentFuncName]; ok {
					// 获取调用变量的路径（如果不存在则使用空字符串）
					callerPath := globalGroupPaths[callerVar]
					groupPaths[paramName] = callerPath
				}
			}
		}

		if !inFunction {
			continue
		}

		// 检查 HTTPServer 赋值
		if matches := httpServerRe.FindStringSubmatch(line); len(matches) > 1 {
			varName := matches[1]
			groupPaths[varName] = ""
		}

		// 检查路由组定义
		if matches := groupRe.FindStringSubmatch(line); len(matches) > 3 {
			newVar := matches[1]
			parentVar := matches[2]
			groupPath := matches[3]

			// 计算累积路径
			parentPath := groupPaths[parentVar]
			fullPath := parentPath + groupPath
			groupPaths[newVar] = fullPath
		}

		// 检查方法定义
		if matches := methodRe.FindStringSubmatch(line); len(matches) > 3 {
			varName := matches[1]
			method := matches[2]
			path := matches[3]

			// 构建完整路径
			basePath := groupPaths[varName]
			fullPath := basePath + path

			// 跳过根路径
			if fullPath == "/" || fullPath == "" {
				continue
			}

			// 提取资源名（从路径中）
			resourceName := extractResourceFromPath(fullPath)
			if resourceName == "" {
				continue
			}

			// 查找或创建路由信息
			route, exists := routeMap[resourceName]
			if !exists {
				route = &RouteInfo{
					Name:       resourceName,
					StructName: toPascalCase(resourceName),
					Comment:    lastComment,
					BasePath:   extractResourceBasePath(fullPath),
				}
				routeMap[resourceName] = route
				lastComment = ""
			}

			// 添加端点信息
			isDetail := strings.Contains(path, ":id") || strings.Contains(path, ":")
			route.Endpoints = append(route.Endpoints, EndpointInfo{
				Method:   method,
				Path:     fullPath,
				IsDetail: isDetail,
			})

			// 设置标志
			switch {
			case method == "POST" && !isDetail:
				route.HasCreate = true
			case method == "GET" && !isDetail:
				route.HasList = true
			case method == "GET" && isDetail:
				route.HasGet = true
			case method == "PUT":
				route.HasUpdate = true
			case method == "DELETE":
				route.HasDelete = true
			}
		}

		// 检查函数结束（简单判断：独立的 } 行）
		if strings.TrimSpace(line) == "}" {
			inFunction = false
			currentFuncName = ""
		}
	}

	// 转换 map 为 slice
	for _, route := range routeMap {
		routes = append(routes, *route)
	}

	return routes
}

// extractResourceFromPath 从路径中提取资源名
func extractResourceFromPath(path string) string {
	// /api/v1/articles/:id -> articles
	// /api/v1/nofly/manage/getNoflyInfo -> nofly_manage
	// /api/v1/auth/login -> auth
	// /api/v1/me -> me
	// /api/v1/drone/health -> health (特殊处理)

	// 特殊处理：如果路径以 /health 结尾，返回 health
	if strings.HasSuffix(path, "/health") {
		return "health"
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")

	var resourceParts []string
	skipNext := false
	// 跳过 api、版本号和服务名（紧跟在版本号后面的部分）
	for i, part := range parts {
		if part == "" || part == "api" {
			continue
		}
		if strings.HasPrefix(part, "v") && len(part) <= 3 {
			skipNext = true // 下一个部分是服务名，跳过
			continue
		}
		if skipNext {
			skipNext = false
			continue // 跳过服务名
		}
		if strings.HasPrefix(part, ":") {
			break
		}
		// 跳过看起来像方法名的部分（以 get/create/update/delete/list 开头）
		lowerPart := strings.ToLower(part)
		if strings.HasPrefix(lowerPart, "get") || strings.HasPrefix(lowerPart, "create") ||
			strings.HasPrefix(lowerPart, "update") || strings.HasPrefix(lowerPart, "delete") ||
			strings.HasPrefix(lowerPart, "list") || strings.HasPrefix(lowerPart, "add") ||
			strings.HasPrefix(lowerPart, "remove") || strings.HasPrefix(lowerPart, "set") {
			break
		}
		_ = i // 避免未使用变量警告
		resourceParts = append(resourceParts, part)
	}

	if len(resourceParts) == 0 {
		return ""
	}
	// 用下划线连接多级资源路径
	return strings.Join(resourceParts, "_")
}

// extractResourceBasePath 提取资源的基础路径（不含参数和方法名部分）
func extractResourceBasePath(path string) string {
	// /api/v1/articles/:id -> /articles
	// /api/v1/drone/nofly/manage/getNoflyInfo -> /nofly/manage
	// /api/v1/auth/login -> /auth

	// 特殊处理：如果路径以 /health 结尾，返回 /health
	if strings.HasSuffix(path, "/health") {
		return "/health"
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	var result []string
	skipNext := false

	for _, part := range parts {
		if part == "" || part == "api" {
			continue
		}
		if strings.HasPrefix(part, "v") && len(part) <= 3 {
			skipNext = true // 下一个部分是服务名，跳过
			continue
		}
		if skipNext {
			skipNext = false
			continue // 跳过服务名
		}
		if strings.HasPrefix(part, ":") {
			break
		}
		// 跳过看起来像方法名的部分
		lowerPart := strings.ToLower(part)
		if strings.HasPrefix(lowerPart, "get") || strings.HasPrefix(lowerPart, "create") ||
			strings.HasPrefix(lowerPart, "update") || strings.HasPrefix(lowerPart, "delete") ||
			strings.HasPrefix(lowerPart, "list") || strings.HasPrefix(lowerPart, "add") ||
			strings.HasPrefix(lowerPart, "remove") || strings.HasPrefix(lowerPart, "set") {
			break
		}
		result = append(result, part)
	}

	if len(result) > 0 {
		return "/" + strings.Join(result, "/")
	}
	return ""
}

// toPascalCase 转换为 PascalCase
func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i := range parts {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	result := strings.Join(parts, "")
	if len(result) > 0 {
		return strings.ToUpper(result[:1]) + result[1:]
	}
	return result
}

// parseTypesFile 解析 types 文件获取字段信息
func parseTypesFile(typesDir, name string) []FieldInfo {
	var fields []FieldInfo

	filename := filepath.Join(typesDir, name+"_types.go")
	content, err := os.ReadFile(filename)
	if err != nil {
		return fields
	}

	// 匹配 CreateXxxRequest 结构体中的字段
	// FieldName Type `json:"jsonName"`
	fieldRe := regexp.MustCompile(`(\w+)\s+(\S+)\s+` + "`" + `json:"(\w+)"` + "`")

	inCreateStruct := false
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.Contains(line, "type Create") && strings.Contains(line, "Request struct") {
			inCreateStruct = true
			continue
		}
		if inCreateStruct {
			if strings.TrimSpace(line) == "}" {
				break
			}
			if matches := fieldRe.FindStringSubmatch(line); len(matches) > 3 {
				fields = append(fields, FieldInfo{
					Name:    matches[1],
					Type:    matches[2],
					JsonTag: matches[3],
				})
			}
		}
	}

	return fields
}

func generateHTTPBookFromRoutes(service, baseURL, port string, routes []RouteInfo) string {
	var sb strings.Builder

	// 查找根路径（服务名路径，如 /drone）和 health 路径
	rootPath := "/"
	healthPath := "/api/v1/health"
	for _, r := range routes {
		if r.Name == service {
			if len(r.Endpoints) > 0 {
				rootPath = r.Endpoints[0].Path
			}
		}
		if r.Name == "health" {
			if len(r.Endpoints) > 0 {
				healthPath = r.Endpoints[0].Path
			}
		}
	}

	// 变量定义（必须在文件开头）
	sb.WriteString(fmt.Sprintf("@baseUrl = %s:%s\n", baseURL, port))
	sb.WriteString("@token = your_jwt_token_here\n")
	for _, r := range routes {
		// 跳过服务名资源（根路径）和 health
		if r.Name == service || r.Name == "health" {
			continue
		}
		varName := toCamelCaseLower(r.Name) + "Id"
		sb.WriteString(fmt.Sprintf("@%s = 1\n", varName))
	}
	sb.WriteString("\n")

	// 头部注释
	sb.WriteString(fmt.Sprintf("# %s HTTP API 调试（Rest Client / httpBook / httpYac 兼容）\n", strings.ToUpper(service[:1])+service[1:]))
	sb.WriteString(fmt.Sprintf("# 启动服务: make run SVC=%s\n", service))
	sb.WriteString("# 使用说明: 先调用 Login 获取 token，然后将 token 值填入上方 @token 变量\n\n")

	// 基础接口 - 使用实际的路径
	sb.WriteString("### Root\n")
	sb.WriteString(fmt.Sprintf("GET {{baseUrl}}%s\n", rootPath))
	sb.WriteString("Accept: application/json\n\n")

	sb.WriteString("### Health\n")
	sb.WriteString(fmt.Sprintf("GET {{baseUrl}}%s\n", healthPath))
	sb.WriteString("Accept: application/json\n\n")

	// 为每个路由生成接口
	for _, r := range routes {
		// 跳过 health 和服务名资源（根路径），因为已经在上面固定生成了
		if r.Name == "health" || r.Name == service {
			continue
		}
		sb.WriteString(generateRouteHTTPBook(r))
	}

	return sb.String()
}

func generateRouteHTTPBook(r RouteInfo) string {
	var sb strings.Builder

	idVar := toCamelCaseLower(r.Name) + "Id"
	comment := strings.TrimSpace(r.Comment)
	if comment == "" {
		comment = r.StructName
	}

	// 如果有端点信息，使用端点信息生成
	if len(r.Endpoints) > 0 {
		for _, ep := range r.Endpoints {
			// 替换路径中的参数为变量
			path := ep.Path
			path = strings.ReplaceAll(path, ":id", "{{"+idVar+"}}")

			// 判断是否需要认证（login 不需要）
			needAuth := !strings.Contains(path, "/login")

			switch ep.Method {
			case "GET":
				// 判断是否为列表接口
				// 1. 路径包含 list/List
				// 2. 资源名以 s 结尾且不是详情接口
				lowerPath := strings.ToLower(ep.Path)
				isList := strings.Contains(lowerPath, "list") || (!ep.IsDetail && strings.HasSuffix(r.Name, "s"))
				if isList {
					sb.WriteString(fmt.Sprintf("### %s - List\n", comment))
					if !strings.Contains(path, "?") {
						path += "?page=1&pageSize=10"
					}
				} else {
					sb.WriteString(fmt.Sprintf("### %s - Get\n", comment))
				}
				sb.WriteString(fmt.Sprintf("GET {{baseUrl}}%s\n", path))
				if needAuth {
					sb.WriteString("Authorization: Bearer {{token}}\n")
				}
				sb.WriteString("Accept: application/json\n\n")
			case "POST":
				sb.WriteString(fmt.Sprintf("### %s - Create\n", comment))
				sb.WriteString(fmt.Sprintf("POST {{baseUrl}}%s\n", path))
				if needAuth {
					sb.WriteString("Authorization: Bearer {{token}}\n")
				}
				sb.WriteString("Content-Type: application/json\n\n")
				sb.WriteString(generateJSONBody(r.Fields, false))
				sb.WriteString("\n\n")
			case "PUT":
				sb.WriteString(fmt.Sprintf("### %s - Update\n", comment))
				sb.WriteString(fmt.Sprintf("PUT {{baseUrl}}%s\n", path))
				if needAuth {
					sb.WriteString("Authorization: Bearer {{token}}\n")
				}
				sb.WriteString("Content-Type: application/json\n\n")
				sb.WriteString(generateJSONBody(r.Fields, true))
				sb.WriteString("\n\n")
			case "DELETE":
				sb.WriteString(fmt.Sprintf("### %s - Delete\n", comment))
				sb.WriteString(fmt.Sprintf("DELETE {{baseUrl}}%s\n", path))
				if needAuth {
					sb.WriteString("Authorization: Bearer {{token}}\n")
				}
				sb.WriteString("Accept: application/json\n\n")
			}
		}
		return sb.String()
	}

	// 回退到旧逻辑（兼容 CRUD 生成的代码）
	// List
	if r.HasList {
		sb.WriteString(fmt.Sprintf("### %s - List\n", comment))
		sb.WriteString(fmt.Sprintf("GET {{baseUrl}}/api/v1%s?page=1&pageSize=10\n", r.BasePath))
		sb.WriteString("Authorization: Bearer {{token}}\n")
		sb.WriteString("Accept: application/json\n\n")
	}

	// Get
	if r.HasGet {
		sb.WriteString(fmt.Sprintf("### %s - Get\n", comment))
		sb.WriteString(fmt.Sprintf("GET {{baseUrl}}/api/v1%s/{{%s}}\n", r.BasePath, idVar))
		sb.WriteString("Authorization: Bearer {{token}}\n")
		sb.WriteString("Accept: application/json\n\n")
	}

	// Create
	if r.HasCreate {
		sb.WriteString(fmt.Sprintf("### %s - Create\n", comment))
		sb.WriteString(fmt.Sprintf("POST {{baseUrl}}/api/v1%s\n", r.BasePath))
		sb.WriteString("Authorization: Bearer {{token}}\n")
		sb.WriteString("Content-Type: application/json\n\n")
		sb.WriteString(generateJSONBody(r.Fields, false))
		sb.WriteString("\n\n")
	}

	// Update
	if r.HasUpdate {
		sb.WriteString(fmt.Sprintf("### %s - Update（只传需要更新的字段）\n", comment))
		sb.WriteString(fmt.Sprintf("PUT {{baseUrl}}/api/v1%s/{{%s}}\n", r.BasePath, idVar))
		sb.WriteString("Authorization: Bearer {{token}}\n")
		sb.WriteString("Content-Type: application/json\n\n")
		sb.WriteString(generateJSONBody(r.Fields, true))
		sb.WriteString("\n\n")
	}

	// Delete
	if r.HasDelete {
		sb.WriteString(fmt.Sprintf("### %s - Delete\n", comment))
		sb.WriteString(fmt.Sprintf("DELETE {{baseUrl}}/api/v1%s/{{%s}}\n", r.BasePath, idVar))
		sb.WriteString("Authorization: Bearer {{token}}\n")
		sb.WriteString("Accept: application/json\n\n")
	}

	return sb.String()
}

func generateJSONBody(fields []FieldInfo, limitFields bool) string {
	if len(fields) == 0 {
		return "{\n  \n}"
	}

	var lines []string
	count := 0
	for _, f := range fields {
		lines = append(lines, fmt.Sprintf("  \"%s\": %s", f.JsonTag, getDefaultValue(f.Type)))
		count++
		if limitFields && count >= 2 {
			break
		}
	}
	return "{\n" + strings.Join(lines, ",\n") + "\n}"
}

func getDefaultValue(goType string) string {
	switch {
	case strings.Contains(goType, "int"):
		return "0"
	case strings.Contains(goType, "float"):
		return "0.0"
	case goType == "bool":
		return "false"
	case strings.Contains(goType, "Time"):
		return `"2024-01-01T00:00:00Z"`
	default:
		return `""`
	}
}

func toCamelCaseLower(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}
