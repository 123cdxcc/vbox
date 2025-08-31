package box

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// 测试辅助函数

// setupTestSSHKey 创建测试用的 SSH 密钥对
func setupTestSSHKey(t *testing.T) (string, func()) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "test_key")
	pubKeyPath := keyPath + ".pub"

	// 生成 SSH 密钥对
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-f", keyPath, "-C", "test@example.com", "-N", "", "-q")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to generate SSH key: %v", err)
	}

	cleanup := func() {
		os.Remove(keyPath)
		os.Remove(pubKeyPath)
	}

	return pubKeyPath, cleanup
}

// cleanupContainer 清理测试容器
func cleanupContainer(t *testing.T, containerName string) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Logf("Failed to create Docker client for cleanup: %v", err)
		return
	}
	defer cli.Close()

	ctx := context.Background()

	// 停止并删除容器
	if err := cli.ContainerStop(ctx, containerName, container.StopOptions{}); err != nil {
		t.Logf("Failed to stop container %s: %v", containerName, err)
	}

	if err := cli.ContainerRemove(ctx, containerName, container.RemoveOptions{Force: true}); err != nil {
		t.Logf("Failed to remove container %s: %v", containerName, err)
	}
}

// verifyDockerPS 使用 docker ps 命令验证容器状态
func verifyDockerPS(t *testing.T, containerName string, shouldExist bool) {
	cmd := exec.Command("docker", "ps", "-a", "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to run docker ps: %v", err)
	}

	containers := strings.Split(strings.TrimSpace(string(output)), "\n")
	found := false
	for _, container := range containers {
		if container == containerName {
			found = true
			break
		}
	}

	if shouldExist && !found {
		t.Errorf("Container %s should exist but was not found in docker ps output", containerName)
	} else if !shouldExist && found {
		t.Errorf("Container %s should not exist but was found in docker ps output", containerName)
	}
}

// verifyDockerImages 使用 docker images 命令验证镜像存在
func verifyDockerImages(t *testing.T, imageName string, shouldExist bool) {
	cmd := exec.Command("docker", "images", "--format", "{{.Repository}}:{{.Tag}}")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to run docker images: %v", err)
	}

	images := strings.Split(strings.TrimSpace(string(output)), "\n")
	found := false
	for _, image := range images {
		if strings.Contains(image, imageName) || image == imageName+":latest" {
			found = true
			break
		}
	}

	if shouldExist && !found {
		t.Errorf("Image %s should exist but was not found in docker images output", imageName)
	} else if !shouldExist && found {
		t.Errorf("Image %s should not exist but was found in docker images output", imageName)
	}
}

// 测试函数

func TestList(t *testing.T) {
	// 创建一个测试容器
	testContainerName := "vbox-test-list"
	defer cleanupContainer(t, testContainerName)

	// 使用 docker run 创建一个测试容器
	cmd := exec.Command("docker", "run", "-d", "--name", testContainerName, "busybox", "sleep", "30")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create test container: %v", err)
	}

	// 测试 List 方法
	containers, err := List()
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	// 验证返回的容器列表中包含我们创建的测试容器
	found := false
	for _, container := range containers {
		if container.Name == testContainerName {
			found = true
			// 验证容器字段
			if container.ID == "" {
				t.Error("Container ID should not be empty")
			}
			if container.Image == "" {
				t.Error("Container Image should not be empty")
			}
			break
		}
	}

	if !found {
		t.Errorf("Test container %s not found in List() results", testContainerName)
	}
}

func TestGet(t *testing.T) {
	// 创建一个测试容器
	testContainerName := "vbox-test-get"
	defer cleanupContainer(t, testContainerName)

	// 使用 docker run 创建一个测试容器
	cmd := exec.Command("docker", "run", "-d", "--name", testContainerName, "busybox", "sleep", "30")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to create test container: %v", err)
	}

	containerID := strings.TrimSpace(string(output))

	// 测试 Get 方法 - 使用完整 ID
	container, err := Get(context.Background(), containerID)
	if err != nil {
		t.Fatalf("Get() with full ID failed: %v", err)
	}

	if container.Name != testContainerName {
		t.Errorf("Expected container name %s, got %s", testContainerName, container.Name)
	}

	// 测试 Get 方法 - 使用短 ID
	shortID := containerID[:12]
	container, err = Get(context.Background(), shortID)
	if err != nil {
		t.Fatalf("Get() with short ID failed: %v", err)
	}

	if container.Name != testContainerName {
		t.Errorf("Expected container name %s, got %s", testContainerName, container.Name)
	}

	// 测试不存在的容器
	_, err = Get(context.Background(), "nonexistent")
	if err != ErrBoxNotFound {
		t.Errorf("Expected ErrBoxNotFound, got %v", err)
	}
}

func TestCreate(t *testing.T) {
	testCases := []struct {
		name        string
		options     CreateOption
		expectError bool
		cleanup     func()
	}{
		{
			name: "Container with SSH port",
			options: CreateOption{
				Name:         "test-create",
				ImageName:    "base",
				ImageVersion: "latest",
				SSHPort:      2222,
				Detached:     true,
				PublicKey:    "/Users/hongfuz/development/code/Golang/vbox/.ssh/demo.pub",
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			containerName := fmt.Sprintf("vbox-%s", tc.options.Name)
			defer cleanupContainer(t, containerName)

			// 执行 Create 方法
			container, err := Create(context.Background(), tc.options)

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Create() failed: %v", err)
			}

			// 验证返回的 Container 对象
			if container == nil {
				t.Fatal("Returned container is nil")
			}

			if container.Name != strings.TrimPrefix(containerName, "vbox-") {
				t.Errorf("Expected container name %s, got %s", containerName, container.Name)
			}

			if container.ID == "" {
				t.Error("Container ID should not be empty")
			}

			// 使用 docker ps 验证容器确实被创建
			verifyDockerPS(t, containerName, true)

			// 验证容器连接到 vbox 网络
			cmd := exec.Command("docker", "inspect", containerName, "--format", "{{range $net, $conf := .NetworkSettings.Networks}}{{$net}}{{end}}")
			output, err := cmd.Output()
			if err != nil {
				t.Fatalf("Failed to inspect container network: %v", err)
			}

			networks := strings.TrimSpace(string(output))
			if !strings.Contains(networks, "vbox-network") {
				t.Errorf("Container should be connected to vbox-network, but networks are: %s", networks)
			}
		})
	}
}
