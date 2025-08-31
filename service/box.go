package service

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"log/slog"

	"github.com/123cdxcc/vbox/box"
	"github.com/123cdxcc/vbox/config"
	"github.com/123cdxcc/vbox/constant"
	"github.com/123cdxcc/vbox/image"
	"github.com/123cdxcc/vbox/pkg/tools"
)

// BoxListParams 包含列出 box 的参数
type BoxListParams struct {
	// 目前不需要额外参数，预留结构体
}

// BoxGetParams 包含获取 box 详细信息的参数
type BoxGetParams struct {
	BoxID string
}

// BoxRunParams 包含运行 box 的参数
type BoxRunParams struct {
	Name      string
	Image     string // 格式: "imageName:version"
	Ports     []box.Port
	SSHPort   int               // SSH 端口映射，0表示随机分配
	PublicKey string            // SSH 公钥内容或文件路径
	Volumes   map[string]string // 卷挂载配置，key为主机路径，value为容器路径
	Detached  bool              // 是否后台运行，默认 true
}

// BoxStopParams 包含停止 box 的参数
type BoxStopParams struct {
	BoxID string
}

// BoxRemoveParams 包含删除 box 的参数
type BoxRemoveParams struct {
	BoxID string
}

// BoxService 提供 box 相关的业务逻辑
type BoxService struct{}

// NewBoxService 创建新的 BoxService 实例
func NewBoxService() *BoxService {
	return &BoxService{}
}

// ListBoxes 列出所有 vbox
func (s *BoxService) ListBoxes(ctx context.Context, params BoxListParams) error {
	containers, err := box.List()
	if err != nil {
		return fmt.Errorf("获取容器列表失败: %v", err)
	}

	if len(containers) == 0 {
		fmt.Println("没有找到任何 vbox")
		return nil
	}

	// 使用 tabwriter 格式化输出
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "BOX ID\tNAME\tIMAGE\tSTATUS\tSTATE")

	for _, container := range containers {
		// 截短 box ID 以便显示
		shortID := container.ID
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			shortID, container.Name, container.Image, container.Status, container.State)
	}
	w.Flush()
	return nil
}

// GetBox 获取特定 box 的详细信息
func (s *BoxService) GetBox(ctx context.Context, params BoxGetParams) error {
	container, err := box.Get(ctx, params.BoxID)
	if err != nil {
		if err == box.ErrBoxNotFound {
			return fmt.Errorf("未找到 ID 为 %s 的 vbox", params.BoxID)
		}
		return fmt.Errorf("获取 box 信息失败: %v", err)
	}

	// 显示详细信息
	fmt.Printf("Box ID: %s\n", container.ID)
	fmt.Printf("名称: %s\n", container.Name)
	fmt.Printf("镜像: %s\n", container.Image)
	fmt.Printf("状态: %s\n", container.Status)
	fmt.Printf("运行状态: %s\n", container.State)

	if len(container.Ports) > 0 {
		fmt.Println("端口映射:")
		for _, port := range container.Ports {
			if port.PublicPort > 0 {
				fmt.Printf("  %d:%d/%s\n", port.PublicPort, port.PrivatePort, port.Type)
			} else {
				fmt.Printf("  %d/%s\n", port.PrivatePort, port.Type)
			}
		}
	}
	return nil
}

// generateRandomPort 生成一个随机可用端口
func (s *BoxService) generateRandomPort() (int, error) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for range 100 { // 最多尝试100次
		port := rng.Intn(65535-1024) + 1024 // 生成1024-65535范围内的端口
		if s.isPortAvailable(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("无法找到可用的随机端口")
}

// isPortAvailable 检查端口是否可用
func (s *BoxService) isPortAvailable(port int) bool {
	ln, err := net.Listen(string(box.PortTypeTCP), fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// Run 运行一个新的 box
func (s *BoxService) Run(ctx context.Context, params BoxRunParams) (boxContainer *box.Container, gerr error) {
	cli := config.GlobalConfig.GetDockerClient()

	// 解析镜像名和版本
	var imageName string
	var imageVersion string

	if !strings.Contains(params.Image, ":") {
		return nil, fmt.Errorf("镜像格式错误: %s，正确格式应为 'name:version'", params.Image)
	}

	parts := strings.SplitN(params.Image, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("镜像格式错误: %s，正确格式应为 'name:version'", params.Image)
	}
	imageName = parts[0]
	imageVersion = parts[1]

	imageFullName := fmt.Sprintf("%s%s:%s", constant.VboxImagePrefix, imageName, imageVersion)

	// 检查镜像是否存在
	exists, err := tools.ImageExists(ctx, cli, imageFullName)
	if err != nil {
		return nil, fmt.Errorf("检查镜像失败: %w", err)
	}

	if !exists {
		// 镜像不存在，尝试从模板构建
		if err := s.buildImageFromTemplate(ctx, imageName, imageVersion); err != nil {
			return nil, fmt.Errorf("镜像 %s 不存在且无法从模板构建: %w", imageFullName, err)
		}
	}

	// 处理SSH端口
	sshPort := params.SSHPort
	if sshPort == 0 {
		// 随机生成端口
		randomPort, err := s.generateRandomPort()
		if err != nil {
			return nil, fmt.Errorf("生成随机SSH端口失败: %w", err)
		}
		sshPort = randomPort
		slog.InfoContext(ctx, fmt.Sprintf("为容器 %s 随机分配SSH端口: %d", params.Name, sshPort))
	}

	// 处理SSH密钥
	var publicKeyPath = params.PublicKey
	if publicKeyPath == "" {
		// 如果没有提供公钥，则生成新的SSH密钥对
		generatedPublicKey, generatedPrivateKey, err := config.GenSSHKeys()
		if err != nil {
			return nil, fmt.Errorf("生成SSH密钥失败: %w", err)
		}
		slog.InfoContext(ctx, fmt.Sprintf("为容器 %s 生成了新的SSH密钥", params.Name))

		// 保存SSH配置和密钥文件
		publicKeyPath, err = config.UpdateSSH(params.Name, "localhost", constant.VboxUser, fmt.Sprintf("%d", sshPort), generatedPrivateKey, generatedPublicKey)
		if err != nil {
			return nil, fmt.Errorf("保存SSH配置失败: %w", err)
		}
		slog.InfoContext(ctx, fmt.Sprintf("为容器 %s 保存了SSH配置", params.Name))
		slog.InfoContext(ctx, "公钥路径", slog.Any("path", publicKeyPath))
	}

	// 创建容器选项
	createOpt := box.CreateOption{
		Name:         params.Name,
		ImageName:    imageName,
		ImageVersion: imageVersion,
		Ports:        params.Ports,
		SSHPort:      sshPort,
		PublicKey:    publicKeyPath,
		Volumes:      params.Volumes,
		Detached:     params.Detached,
	}

	// 调用 box.Create 创建容器
	container, err := box.Create(ctx, createOpt)
	if err != nil {
		return nil, err
	}

	return container, nil
}

// buildImageFromTemplate 根据模板构建镜像
func (s *BoxService) buildImageFromTemplate(ctx context.Context, imageName, imageVersion string) error {
	templatePath, err := tools.GetTemplatePath(config.GlobalConfig.TemplatesDirPath, imageName)
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, fmt.Sprintf("正在构建镜像 %s，使用模板 %s...", imageName, templatePath))

	ch, err := image.BuildFromDockerfile(ctx, templatePath, imageName, imageVersion)
	if err != nil {
		return err
	}
	for buildResponse := range ch {
		if buildResponse.Error != nil {
			return fmt.Errorf("镜像构建失败: %s", *buildResponse.Error)
		}
		slog.InfoContext(ctx, buildResponse.Stream)
	}
	return nil
}

// StopBox 停止指定的 box
func (s *BoxService) StopBox(ctx context.Context, params BoxStopParams) error {
	if err := box.Stop(ctx, params.BoxID); err != nil {
		return fmt.Errorf("停止 box 失败: %v", err)
	}
	return nil
}

// RemoveBox 停止并删除指定的 box
func (s *BoxService) RemoveBox(ctx context.Context, params BoxRemoveParams) error {
	// 先获取容器信息，用于获取容器名称
	container, err := box.Get(ctx, params.BoxID)
	if err != nil {
		if err == box.ErrBoxNotFound {
			return fmt.Errorf("未找到 ID 为 %s 的 vbox", params.BoxID)
		}
		// 如果获取容器信息失败，仍然尝试删除容器
		slog.InfoContext(ctx, fmt.Sprintf("获取容器信息失败，仍将尝试删除容器: %v", err))
	}

	// 删除容器
	if err := box.StopAndDelete(ctx, params.BoxID); err != nil {
		return fmt.Errorf("删除 box 失败: %v", err)
	}

	// 如果成功获取了容器信息，则删除对应的SSH配置
	if container != nil {
		if err := config.RemoveSSH(container.Name); err != nil {
			slog.InfoContext(ctx, fmt.Sprintf("删除SSH配置失败: %v", err))
			// 这里不返回错误，因为容器已经删除成功，只是SSH配置删除失败
		} else {
			slog.InfoContext(ctx, fmt.Sprintf("已删除容器 %s 的SSH配置", container.Name))
		}
	}

	return nil
}
