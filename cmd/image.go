/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/123cdxcc/vbox/service"
	"github.com/spf13/cobra"
)

var imageService = service.NewImageService()

// imageCmd represents the image command
var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "管理 box 镜像",
	Long:  `管理 box 镜像的工具，包括构建和列出镜像。`,
}

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "从 Dockerfile 构建镜像",
	Long:  `从指定的 Dockerfile 构建 vbox 镜像。`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		// 获取flag值
		dockerfile, _ := cmd.Flags().GetString("dockerfile")
		name, _ := cmd.Flags().GetString("name")
		version, _ := cmd.Flags().GetString("version")

		params := service.ImageBuildParams{
			DockerfilePath: dockerfile,
			Name:           name,
			Version:        version,
		}

		if err := imageService.BuildImage(ctx, params); err != nil {
			fmt.Printf("错误：%v\n", err)
			os.Exit(1)
		}
	},
}

// imageListCmd represents the list command
var imageListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有 box 镜像",
	Long:  `列出所有 box 镜像。`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		params := service.ImageListParams{}

		if err := imageService.ListImages(ctx, params); err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
	},
}

// imagesCmd represents the images command (shortcut for image list)
var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "列出所有 box 镜像",
	Long:  `列出所有 box 镜像`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		imageService := service.NewImageService()
		params := service.ImageListParams{}

		if err := imageService.ListImages(ctx, params); err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
	},
}

// rmiCmd represents the rmi command
var rmiCmd = &cobra.Command{
	Use:   "rmi <image-id>",
	Short: "删除 box 镜像",
	Long:  `根据镜像ID删除指定的 box 镜像。`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		params := service.ImageRmiParams{
			ImageID: args[0],
			Force:   false,
		}

		if params.ImageID == "" {
			fmt.Printf("错误：镜像ID不能为空\n")
			os.Exit(1)
		}

		if err := imageService.RmiImage(ctx, params); err != nil {
			fmt.Printf("错误：%v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(imageCmd)
	rootCmd.AddCommand(rmiCmd)
	rootCmd.AddCommand(imagesCmd)

	// 添加子命令
	imageCmd.AddCommand(buildCmd)
	imageCmd.AddCommand(imageListCmd)
	imageCmd.AddCommand(rmiCmd)

	// 为build命令添加flags
	buildCmd.Flags().StringP("dockerfile", "f", "", "Dockerfile 的路径 (必需)")
	buildCmd.Flags().StringP("name", "n", "", "镜像名称 (必需)")
	buildCmd.Flags().StringP("version", "v", "", "镜像版本 (必需)")

	// 标记为必需参数
	buildCmd.MarkFlagRequired("dockerfile")
	buildCmd.MarkFlagRequired("name")
	buildCmd.MarkFlagRequired("version")

	// 为rmi命令添加flags
	rmiCmd.Flags().BoolP("force", "f", false, "强制删除")
}
