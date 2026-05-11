package cmd

import (
	"fmt"

	"github.com/goupter/goupter/cmd/goupter/internal/gen"
	"github.com/spf13/cobra"
)

var genModelCmd = &cobra.Command{
	Use:   "model",
	Short: "从数据库表生成GORM模型",
	Long: `从MySQL数据库表生成GORM模型代码。

生成的模型包含:
  - 结构体定义（带GORM标签）
  - 标准CRUD接口实现
  - 分页查询方法
  - 事务支持

示例:
  # 从数据库生成所有表的模型
  goupter gen model --dsn "user:pass@tcp(localhost:3306)/dbname"

  # 生成指定表的模型
  goupter gen model --dsn "user:pass@tcp(localhost:3306)/dbname" --tables users,orders

  # 指定输出目录
  goupter gen model --dsn "user:pass@tcp(localhost:3306)/dbname" --out ./model`,
	Run: runGenModel,
}

var (
	dsn         string
	tables      string
	outDir      string
	packageName string
	withQuery   bool
)

func init() {
	genCmd.AddCommand(genModelCmd)

	genModelCmd.Flags().StringVar(&dsn, "dsn", "", "数据库连接字符串 (必需)")
	genModelCmd.Flags().StringVar(&tables, "tables", "", "要生成的表名，多个表用逗号分隔 (留空则生成所有表)")
	genModelCmd.Flags().StringVarP(&outDir, "out", "o", "./model", "输出目录")
	genModelCmd.Flags().StringVarP(&packageName, "package", "p", "model", "包名")
	genModelCmd.Flags().BoolVar(&withQuery, "with-query", true, "是否生成查询方法")

	genModelCmd.MarkFlagRequired("dsn")
}

func runGenModel(cmd *cobra.Command, args []string) {
	generator, err := gen.NewModelGenerator(gen.ModelConfig{
		DSN:         dsn,
		Tables:      tables,
		OutDir:      outDir,
		PackageName: packageName,
		WithQuery:   withQuery,
	})
	if err != nil {
		exitWithError("初始化生成器失败", err)
	}
	defer generator.Close()

	if err := generator.Generate(); err != nil {
		exitWithError("生成模型失败", err)
	}

	fmt.Println("✅ 模型生成成功!")
}
