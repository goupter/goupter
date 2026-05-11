package cmd

import (
	"github.com/spf13/cobra"
)

var genCmd = &cobra.Command{
	Use:     "gen",
	Aliases: []string{"generate", "g"},
	Short:   "生成代码",
	Long: `生成各种代码，包括:
  - model: 从数据库表生成GORM模型
  - service: 生成Service层代码
  - handler: 生成Handler层代码
  - api: 生成完整的API代码（包含model、service、handler）`,
}

func init() {
	rootCmd.AddCommand(genCmd)
}
