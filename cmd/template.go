package cmd

import (
	"fmt"
	"os"

	"github.com/123cdxcc/vbox/config"
	template "github.com/123cdxcc/vbox/env"
	"github.com/spf13/cobra"
)

// templateCmd represents the template command
var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "管理 box 模板",
	Long:  `管理 box 模板的工具，包括初始化模板。`,
}

// templateInitCmd represents the init command
var templateInitCmd = &cobra.Command{
	Use:   "init",
	Short: "初始化模板目录",
	Long:  `初始化模板目录，创建默认的环境模板和配置文件。`,
	Run: func(cmd *cobra.Command, args []string) {
		// 获取目标目录
		targetDir := config.GlobalConfig.TemplatesDirPath

		fmt.Printf("正在初始化模板到目录: %s\n", targetDir)

		if err := template.Init(targetDir); err != nil {
			fmt.Printf("错误：%v\n", err)
			os.Exit(1)
		}

		fmt.Printf("模板初始化完成！\n")
	},
}

func init() {
	rootCmd.AddCommand(templateCmd)

	// 添加子命令
	templateCmd.AddCommand(templateInitCmd)
}
