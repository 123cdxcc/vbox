/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/123cdxcc/vbox/box"
	"github.com/123cdxcc/vbox/service"

	"github.com/spf13/cobra"
)

var boxService = service.NewBoxService()

// boxCmd represents the box command
var boxCmd = &cobra.Command{
	Use:   "box",
	Short: "管理 box",
	Long:  `box 管理。`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有 box",
	Long:  `列出所有 box`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		params := service.BoxListParams{}

		if err := boxService.ListBoxes(ctx, params); err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
	},
}

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get <box-id>",
	Short: "获取特定 box 的详细信息",
	Long:  `根据 box ID 获取单个 box 的详细信息`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		params := service.BoxGetParams{
			BoxID: args[0],
		}

		if err := boxService.GetBox(ctx, params); err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
	},
}

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run [OPTIONS] IMAGE",
	Short: "运行一个新的 box",
	Long:  `运行一个新的 box`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		// 获取命令行参数
		name, _ := cmd.Flags().GetString("name")
		portMappings, _ := cmd.Flags().GetStringSlice("port")
		sshPort, _ := cmd.Flags().GetInt("ssh-port")
		publicKey, _ := cmd.Flags().GetString("public-key")
		volumeMappings, _ := cmd.Flags().GetStringSlice("volume")
		detach, _ := cmd.Flags().GetBool("detach")

		// 解析端口映射
		ports, err := parsePorts(portMappings)
		if err != nil {
			fmt.Printf("端口映射解析错误: %v\n", err)
			os.Exit(1)
		}

		// 解析卷映射
		volumes, err := parseVolumes(volumeMappings)
		if err != nil {
			fmt.Printf("卷映射解析错误: %v\n", err)
			os.Exit(1)
		}

		// 如果没有指定名称，生成一个随机名称
		if name == "" {
			fmt.Printf("错误: 必须指定 box 名称\n")
			os.Exit(1)
		}

		params := service.BoxRunParams{
			Name:      name,
			Image:     args[0], // 镜像名称:版本
			Ports:     ports,
			SSHPort:   sshPort,
			PublicKey: publicKey,
			Volumes:   volumes,
			Detached:  detach,
		}

		container, err := boxService.Run(ctx, params)
		if err != nil {
			fmt.Printf("运行 box 失败: %v\n", err)
			os.Exit(1)
		}

		if detach {
			fmt.Printf("成功创建 box: %s (ID: %s)\n", container.Name, container.ID)
			if sshPort > 0 || len(ports) > 0 {
				fmt.Println("端口映射:")
				for _, port := range container.Ports {
					if port.PublicPort > 0 {
						fmt.Printf("  %d:%d/%s\n", port.PublicPort, port.PrivatePort, port.Type)
					}
				}
			}
		}
	},
}

// parsePorts 解析端口映射参数
func parsePorts(portMappings []string) ([]box.Port, error) {
	var ports []box.Port

	for _, mapping := range portMappings {
		// 支持格式: "8080:80", "8080:80/tcp", "80"
		parts := strings.Split(mapping, ":")
		var hostPort, containerPort int
		var protocol string = "tcp"

		if len(parts) == 1 {
			// 格式: "80" 或 "80/tcp"
			portAndProtocol := strings.Split(parts[0], "/")
			var err error
			containerPort, err = strconv.Atoi(portAndProtocol[0])
			if err != nil {
				return nil, fmt.Errorf("无效的端口号: %s", portAndProtocol[0])
			}
			hostPort = 0 // 随机分配主机端口
			if len(portAndProtocol) > 1 {
				protocol = portAndProtocol[1]
			}
		} else if len(parts) == 2 {
			// 格式: "8080:80" 或 "8080:80/tcp"
			var err error
			hostPort, err = strconv.Atoi(parts[0])
			if err != nil {
				return nil, fmt.Errorf("无效的主机端口号: %s", parts[0])
			}

			portAndProtocol := strings.Split(parts[1], "/")
			containerPort, err = strconv.Atoi(portAndProtocol[0])
			if err != nil {
				return nil, fmt.Errorf("无效的容器端口号: %s", portAndProtocol[0])
			}
			if len(portAndProtocol) > 1 {
				protocol = portAndProtocol[1]
			}
		} else {
			return nil, fmt.Errorf("无效的端口映射格式: %s", mapping)
		}

		// 验证协议
		if protocol != "tcp" && protocol != "udp" {
			return nil, fmt.Errorf("不支持的协议: %s", protocol)
		}

		ports = append(ports, box.Port{
			PrivatePort: containerPort,
			PublicPort:  hostPort,
			Type:        box.PortType(protocol),
		})
	}

	return ports, nil
}

// parseVolumes 解析卷映射参数
func parseVolumes(volumeMappings []string) (map[string]string, error) {
	volumes := make(map[string]string)

	for _, mapping := range volumeMappings {
		// 支持格式: "/host/path:/container/path"
		parts := strings.Split(mapping, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("无效的卷映射格式: %s，正确格式为 'host_path:container_path'", mapping)
		}

		hostPath := strings.TrimSpace(parts[0])
		containerPath := strings.TrimSpace(parts[1])

		if hostPath == "" || containerPath == "" {
			return nil, fmt.Errorf("主机路径和容器路径都不能为空: %s", mapping)
		}

		volumes[hostPath] = containerPath
	}

	return volumes, nil
}

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop <box-id>",
	Short: "停止运行中的 box",
	Long:  `根据 box ID 停止运行中的 box`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		params := service.BoxStopParams{
			BoxID: args[0],
		}

		if err := boxService.StopBox(ctx, params); err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}

		fmt.Printf("成功停止 box: %s\n", params.BoxID)
	},
}

// rmCmd represents the rm command
var rmCmd = &cobra.Command{
	Use:   "rm <box-id>",
	Short: "停止并删除 box",
	Long:  `根据 box ID 停止并删除指定的 box`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		params := service.BoxRemoveParams{
			BoxID: args[0],
		}

		if err := boxService.RemoveBox(ctx, params); err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}

		fmt.Printf("成功删除 box: %s\n", params.BoxID)
	},
}

func init() {
	rootCmd.AddCommand(boxCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(rmCmd)

	boxCmd.AddCommand(listCmd)
	boxCmd.AddCommand(getCmd)
	boxCmd.AddCommand(runCmd)
	boxCmd.AddCommand(stopCmd)
	boxCmd.AddCommand(rmCmd)

	// 为 run 命令添加 flags
	runCmd.Flags().StringP("name", "", "", "指定容器名称")
	runCmd.Flags().StringSliceP("port", "p", []string{}, "端口映射 (格式: host_port:container_port[/protocol])")
	runCmd.Flags().IntP("ssh-port", "", 0, "SSH 端口映射 (0 表示随机分配)")
	runCmd.Flags().StringP("public-key", "", "", "SSH 公钥文件路径或公钥内容")
	runCmd.Flags().StringSliceP("volume", "v", []string{}, "卷映射 (格式: host_path:container_path)")
	runCmd.Flags().BoolP("detach", "d", true, "后台运行容器")
}
