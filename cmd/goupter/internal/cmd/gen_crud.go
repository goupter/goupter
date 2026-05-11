package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/goupter/goupter/cmd/goupter/internal/gen"
	"github.com/spf13/cobra"
)

var genCRUDCmd = &cobra.Command{
	Use:   "crud",
	Short: "生成 CRUD 代码（Model + Handler + Routes）",
	Long: `从数据库表生成完整的 CRUD 代码，包括：
  - Model: GORM 模型（./model/）
  - Handler: HTTP 处理器（./handler/）
  - Routes: 路由注册（./handler/routes.go）

使用 --service 生成到服务目录：
  - Routes: cmd/<service-name>/routes.go（package main）
  - Handler: cmd/<service-name>/handler/<table>_handler.go（package handler）

示例:
  # 生成所有表的 CRUD
  goupter gen crud --dsn "user:pass@tcp(localhost:3306)/dbname"

  # 生成指定表
  goupter gen crud --dsn "user:pass@tcp(localhost:3306)/dbname" --tables users,orders

  # 指定输出目录
  goupter gen crud --dsn "..." --model-dir ./model --handler-dir ./handler

  # 生成到指定服务目录
  goupter gen crud --dsn "..." --service user-service`,
	Run: runGenCRUD,
}

var (
	crudDSN        string
	crudTables     string
	crudModelDir   string
	crudHandlerDir string
	crudModuleName string
	crudService    string
	crudTrimPrefix string
)

func init() {
	genCmd.AddCommand(genCRUDCmd)

	genCRUDCmd.Flags().StringVar(&crudDSN, "dsn", "", "数据库连接字符串 (必需)")
	genCRUDCmd.Flags().StringVar(&crudTables, "tables", "", "要生成的表名，逗号分隔")
	genCRUDCmd.Flags().StringVar(&crudModelDir, "model-dir", "./model", "Model 输出目录")
	genCRUDCmd.Flags().StringVar(&crudHandlerDir, "handler-dir", "./handler", "Handler 输出目录")
	genCRUDCmd.Flags().StringVar(&crudModuleName, "module", "", "Go 模块名（自动从 go.mod 读取）")
	genCRUDCmd.Flags().StringVar(&crudService, "service", "", "目标服务名（生成到 cmd/<service>/...）")
	genCRUDCmd.Flags().StringVar(&crudTrimPrefix, "trim-prefix", "", "过滤表名前缀（如 busi_）")

	genCRUDCmd.MarkFlagRequired("dsn")
}

var serviceNameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9._-]*$`)

func runGenCRUD(cmd *cobra.Command, args []string) {
	crudService = strings.TrimSpace(crudService)

	// 校验服务名
	if crudService != "" {
		if !serviceNameRegex.MatchString(crudService) {
			exitWithError("无效的服务名", fmt.Errorf("服务名只能包含字母、数字、点、下划线和连字符，且必须以字母开头"))
		}
		// 检查参数冲突
		if cmd.Flags().Changed("handler-dir") {
			fmt.Println("⚠️  警告: --service 已指定，--handler-dir 将被忽略")
		}
	}

	// 自动读取模块名
	if crudModuleName == "" {
		crudModuleName = readGoModuleName()
		if crudModuleName == "" {
			exitWithError("无法读取模块名", fmt.Errorf("请指定 --module 或确保当前目录有 go.mod"))
		}
	}

	handlerDir := crudHandlerDir
	typesDir := "./types"
	routesFile := filepath.Join(crudHandlerDir, "routes.go")
	if crudService != "" {
		handlerDir = filepath.Join("cmd", crudService, "handler")
		typesDir = filepath.Join("cmd", crudService, "types")
		routesFile = filepath.Join("cmd", crudService, "routes.go")
	}

	config := gen.CRUDConfig{
		ModelConfig: gen.ModelConfig{
			DSN:         crudDSN,
			Tables:      crudTables,
			OutDir:      crudModelDir,
			PackageName: "model",
			TrimPrefix:  crudTrimPrefix,
		},
		ServiceName: crudService,
		TrimPrefix:  crudTrimPrefix,
		HandlerDir:  handlerDir,
		TypesDir:    typesDir,
		RoutesFile:  routesFile,
		ModuleName:  crudModuleName,
	}

	generator, err := gen.NewCRUDGenerator(config)
	if err != nil {
		exitWithError("初始化生成器失败", err)
	}
	defer generator.Close()

	if err := generator.Generate(); err != nil {
		exitWithError("生成 CRUD 失败", err)
	}

	fmt.Println("\n✅ CRUD 生成成功!")
	fmt.Println("\n使用方式:")
	fmt.Println("  在 main.go 中调用各表的路由注册函数:")
	if crudService != "" {
		fmt.Println("    RegisterUserRoutes(api, db)")
		fmt.Println("    RegisterOrderRoutes(api, db)")
	} else {
		fmt.Println("    handler.RegisterUserRoutes(api, db)")
		fmt.Println("    handler.RegisterOrderRoutes(api, db)")
	}
}

func readGoModuleName() string {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module ")
		}
	}
	return ""
}
