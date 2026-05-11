package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version 版本号
	Version = "v0.0.1"
	// BuildTime 构建时间
	BuildTime = ""
	// GitCommit Git提交
	GitCommit = ""
)

var rootCmd = &cobra.Command{
	Use:   "goupter",
	Short: "Goupter - 一个轻量级的Go微服务框架CLI工具",
	Long: `Goupter CLI 是一个代码生成工具，用于快速创建基于Goupter框架的微服务项目。

命令:
  new       创建新的微服务（支持多服务共享 go.mod）
  gen       代码生成命令组
    model     从数据库表生成 GORM 模型
    crud      生成完整 CRUD 代码（Model + Handler + Routes）
    httpbook  生成 HTTP API 调试文件（.http 格式）

示例:
  goupter new user                                    # 创建 user 服务
  goupter gen model --dsn "user:pass@tcp(...)/db"    # 生成模型
  goupter gen crud --dsn "..." --service user        # 生成 CRUD 到服务目录
  goupter gen httpbook --service user                # 生成 HTTP 调试文件`,
	Version: Version,
}

// Execute 执行根命令
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.SetVersionTemplate(fmt.Sprintf("Goupter CLI v%s\nBuild: %s\nCommit: %s\n", Version, BuildTime, GitCommit))

	// 添加全局标志
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "显示详细输出")
}

// exitWithError 错误退出
func exitWithError(msg string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %s: %v\n", msg, err)
	} else {
		fmt.Fprintf(os.Stderr, "错误: %s\n", msg)
	}
	os.Exit(1)
}
