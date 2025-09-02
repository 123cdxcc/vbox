package image

import (
	"context"
	"fmt"
	"testing"
)

// TestBuildFromDockerfile 测试从单个 Dockerfile 构建
func TestBuildFromDockerfile(t *testing.T) {
	ch, err := Build(
		context.Background(),
		BuildOptions{
			Name:           "golang",
			Version:        "1.25.0",
			Dockerfile:     "/Users/hongfuz/development/code/Golang/vbox/env/Dockerfile",
			SetupScript:    "/Users/hongfuz/development/code/Golang/vbox/env/setup.sh",
			SetupEnvScript: "/Users/hongfuz/development/code/Golang/vbox/env/template/golang/1.25.0.sh",
			Remove:         true,
			NoCache:        false,
		},
	)
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}
	for resp := range ch {
		if resp.Error != nil {
			t.Fatalf("构建失败: %v", resp.ErrorDetail.Message)
		}
		fmt.Println(resp.Stream)
	}
}

// TestBuildWithContext 测试带构建上下文的构建
func TestBuildWithContext(t *testing.T) {
	opts := BuildOptions{
		Name:       "vbox-app",
		Version:    "latest",
		Dockerfile: "/Users/hongfuz/development/code/Golang/vbox/template/Base-Dockerfile",
		Remove:     true,
		NoCache:    false,
	}

	ch, err := Build(context.Background(), opts)
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}
	for resp := range ch {
		if resp.Error != nil {
			t.Fatalf("构建失败: %v", resp.ErrorDetail.Message)
		}
		fmt.Println(resp.Stream)
	}
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}
}

// TestList 测试列出 vbox 镜像
func TestList(t *testing.T) {
	images, err := List(context.Background())
	if err != nil {
		t.Fatalf("获取镜像列表失败: %v", err)
	}

	fmt.Printf("找到 %d 个 vbox 镜像:\n", len(images))
	for _, img := range images {
		fmt.Printf("ID: %s, 名称: %s, 版本: %s, 大小: %d, 创建时间: %s\n",
			img.ID[:12], // 只显示前12个字符
			img.Name,
			img.Version,
			img.Size,
			img.Created.Format("2006-01-02 15:04:05"),
		)
	}
}
