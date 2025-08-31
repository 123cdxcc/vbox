package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/123cdxcc/vbox/image"
)

// ImageBuildParams 包含构建镜像的参数
type ImageBuildParams struct {
	DockerfilePath string
	Name           string
	Version        string
}

// ImageListParams 包含列出镜像的参数
type ImageListParams struct {
	// 目前不需要额外参数，预留结构体
}

// ImageRmiParams 包含删除镜像的参数
type ImageRmiParams struct {
	ImageID string
	Force   bool
}

// ImageService 提供 image 相关的业务逻辑
type ImageService struct{}

// NewImageService 创建新的 ImageService 实例
func NewImageService() *ImageService {
	return &ImageService{}
}

// BuildImage 从 Dockerfile 构建镜像
func (s *ImageService) BuildImage(ctx context.Context, params ImageBuildParams) error {
	// 检查 Dockerfile 是否存在
	if _, err := os.Stat(params.DockerfilePath); os.IsNotExist(err) {
		return fmt.Errorf("Dockerfile 不存在: %s", params.DockerfilePath)
	}

	// 转换为绝对路径
	absPath, err := filepath.Abs(params.DockerfilePath)
	if err != nil {
		return fmt.Errorf("无法获取绝对路径: %v", err)
	}

	fmt.Printf("开始构建镜像: %s:%s\n", params.Name, params.Version)
	fmt.Printf("使用 Dockerfile: %s\n", absPath)

	respCh, err := image.BuildFromDockerfile(ctx, absPath, params.Name, params.Version)
	if err != nil {
		return fmt.Errorf("构建失败: %v", err)
	}

	// 处理构建响应
	for resp := range respCh {
		if resp.Error != nil {
			return fmt.Errorf("构建错误: %s", *resp.Error)
		}
		if resp.ErrorDetail.Message != "" {
			return fmt.Errorf("构建错误: %s", resp.ErrorDetail.Message)
		}
		if resp.Stream != "" {
			// 去掉末尾的换行符
			output := strings.TrimSuffix(resp.Stream, "\n")
			if output != "" {
				fmt.Print(output)
			}
		}
	}

	fmt.Printf("\n镜像构建完成: vbox-%s:%s\n", params.Name, params.Version)
	return nil
}

// ListImages 列出所有 vbox 镜像
func (s *ImageService) ListImages(ctx context.Context, params ImageListParams) error {
	images, err := image.List(ctx)
	if err != nil {
		return fmt.Errorf("获取镜像列表失败: %v", err)
	}

	if len(images) == 0 {
		fmt.Println("未找到任何 vbox 镜像")
		return nil
	}

	// 使用 tabwriter 格式化输出
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tVERSION\tIMAGE ID\tSIZE\tCREATED")

	for _, img := range images {
		// 格式化大小
		sizeStr := s.formatSize(img.Size)

		// 格式化创建时间
		createdStr := img.Created.Format(time.DateTime)

		// 截断镜像ID到12个字符
		shortID := img.ID
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			img.Name, img.Version, shortID, sizeStr, createdStr)
	}
	w.Flush()
	return nil
}

// formatSize 格式化字节大小为人类可读的格式
func (s *ImageService) formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// RmiImage 删除指定的镜像
func (s *ImageService) RmiImage(ctx context.Context, params ImageRmiParams) error {
	fmt.Printf("正在删除镜像: %s\n", params.ImageID)

	if err := image.Delete(ctx, params.ImageID, params.Force); err != nil {
		return fmt.Errorf("删除镜像失败: %v", err)
	}

	fmt.Printf("镜像删除成功: %s\n", params.ImageID)
	return nil
}
