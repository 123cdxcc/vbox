package image

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/123cdxcc/vbox/config"
	"github.com/123cdxcc/vbox/constant"
	"github.com/123cdxcc/vbox/pkg/tools"
	"github.com/moby/moby/api/types/build"
	"github.com/moby/moby/api/types/image"
)

// BuildOptions 构建选项
type BuildOptions struct {
	Name        string             // 镜像名称
	Version     string             // 版本
	Dockerfile  string             // Dockerfile 路径
	Remove      bool               // 是否删除中间容器
	ForceRemove bool               // 强制删除中间容器
	NoCache     bool               // 不使用缓存
	BuildArgs   map[string]*string // 构建参数
}

type BuildResponse struct {
	Stream      string `json:"stream"`
	ErrorDetail struct {
		Message string `json:"message"`
	} `json:"errorDetail"`
	Error *string `json:"error"`
}

// Build 构建 Docker 镜像
func Build(ctx context.Context, opts BuildOptions) (<-chan BuildResponse, error) {
	cli := config.GlobalConfig.GetDockerClient()

	// 创建 tar 构建上下文
	buildContext, err := tools.CreateBuildContext(opts.Dockerfile)
	if err != nil {
		return nil, fmt.Errorf("创建构建上下文失败: %w", err)
	}
	defer buildContext.Close()

	// 准备构建选项
	buildOptions := build.ImageBuildOptions{
		Tags:        []string{fmt.Sprintf("%s%s:%s", constant.VboxImagePrefix, opts.Name, opts.Version)},
		Dockerfile:  filepath.Base(opts.Dockerfile),
		Remove:      opts.Remove,
		ForceRemove: opts.ForceRemove,
		NoCache:     opts.NoCache,
		BuildArgs:   opts.BuildArgs,
	}

	// 执行构建
	resp, err := cli.ImageBuild(ctx, buildContext, buildOptions)
	if err != nil {
		return nil, fmt.Errorf("构建镜像失败: %w", err)
	}
	ch := make(chan BuildResponse)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			var resp BuildResponse
			if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
				return
			}
			ch <- resp
		}
	}()

	return ch, nil
}

// BuildFromDockerfile 从单个 Dockerfile 构建镜像（简化版本）
func BuildFromDockerfile(ctx context.Context, dockerfilePath string, name, version string) (<-chan BuildResponse, error) {
	return Build(ctx, BuildOptions{
		Name:       name,
		Version:    version,
		Dockerfile: dockerfilePath,
		Remove:     true,
		NoCache:    false,
	})
}

type Image struct {
	ID      string    // 镜像ID
	Name    string    // 名称
	Version string    // 版本
	Size    int64     // 镜像大小
	Created time.Time // 创建时间
}

func List(ctx context.Context) ([]Image, error) {
	cli := config.GlobalConfig.GetDockerClient()

	// 获取所有镜像
	images, err := cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取镜像列表失败: %w", err)
	}

	var vboxImages []Image
	for _, img := range images {
		// 检查镜像的标签是否以 "vbox-" 开头
		for _, tag := range img.RepoTags {
			// 检查标签是否以 "vbox-" 开头
			if strings.HasPrefix(tag, "vbox-") {
				// 解析标签，格式为 "vbox-name:version"
				parts := strings.Split(tag, ":")
				if len(parts) != 2 {
					continue // 跳过格式不正确的标签
				}

				fullName := parts[0] // vbox-name
				version := parts[1]  // version

				// 提取名称部分（去掉 "vbox-" 前缀）
				name := strings.TrimPrefix(fullName, "vbox-")

				// 转换创建时间
				createdTime := time.Unix(img.Created, 0)

				// 去掉 ID 中的 "sha256:" 前缀
				imageID := strings.TrimPrefix(img.ID, "sha256:")

				vboxImage := Image{
					ID:      imageID,
					Name:    name,
					Version: version,
					Size:    img.Size,
					Created: createdTime,
				}
				vboxImages = append(vboxImages, vboxImage)
			}
		}
	}

	return vboxImages, nil
}

// Delete 删除指定的镜像
// imageID 必须是镜像ID（可以是完整ID或短ID）
// force 参数决定是否强制删除被容器使用的镜像
func Delete(ctx context.Context, imageID string, force bool) error {
	cli := config.GlobalConfig.GetDockerClient()

	// 确保镜像ID包含 sha256: 前缀（如果没有的话）
	if !strings.HasPrefix(imageID, "sha256:") {
		imageID = "sha256:" + imageID
	}

	// 删除镜像
	_, err := cli.ImageRemove(ctx, imageID, image.RemoveOptions{
		Force:         force, // 是否强制删除被容器使用的镜像
		PruneChildren: true,  // 删除未标记的父镜像
	})
	if err != nil {
		return fmt.Errorf("删除镜像失败: %w", err)
	}

	return nil
}
