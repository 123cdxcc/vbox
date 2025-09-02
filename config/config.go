package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/moby/moby/client"
)

type Config struct {
	AppConfigDirPath string
	AppSSHConfigPath string
	AppSSHDirPath    string
	TemplatesDirPath string
	DockerClient     *client.Client
}

var GlobalConfig *Config

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	appConfigDirPath := filepath.Join(userHomeDir, ".config", "vbox")
	appSSHDirPath := filepath.Join(appConfigDirPath, "ssh")
	GlobalConfig = &Config{
		AppConfigDirPath: appConfigDirPath,
		AppSSHDirPath:    appSSHDirPath,
		AppSSHConfigPath: filepath.Join(appSSHDirPath, "config"),
		TemplatesDirPath: filepath.Join(appConfigDirPath, "env"),
	}
	userHomeSSHConfigPath := filepath.Join(userHomeDir, ".ssh", "config")

	// 确保SSH配置目录存在
	if err := os.MkdirAll(filepath.Dir(GlobalConfig.AppSSHConfigPath), 0700); err != nil {
		panic(fmt.Errorf("failed to create SSH config directory: %w", err))
	}

	// 检查并创建vbox SSH配置文件
	if _, err := os.Stat(GlobalConfig.AppSSHConfigPath); os.IsNotExist(err) {
		// 创建空的配置文件
		if err := os.WriteFile(GlobalConfig.AppSSHConfigPath, []byte(""), 0600); err != nil {
			panic(fmt.Errorf("failed to create vbox SSH config file: %w", err))
		}
	}

	// 构建vbox配置文件的绝对路径用于Include语句
	includeStatement := "Include " + GlobalConfig.AppSSHConfigPath

	// 确保主SSH配置目录存在
	if err := os.MkdirAll(filepath.Dir(userHomeSSHConfigPath), 0700); err != nil {
		panic(fmt.Errorf("failed to create main SSH config directory: %w", err))
	}

	// 检查主配置文件是否存在，不存在则创建
	if _, err := os.Stat(userHomeSSHConfigPath); os.IsNotExist(err) {
		// 创建新的配置文件，第一行就是Include指令
		if err := os.WriteFile(userHomeSSHConfigPath, []byte(includeStatement+"\n"), 0600); err != nil {
			panic(fmt.Errorf("failed to create main SSH config file: %w", err))
		}
	} else {
		// 文件存在，检查是否包含Include指令
		needsInclude, err := checkAndAddIncludeStatement(userHomeSSHConfigPath, includeStatement)
		if err != nil {
			panic(fmt.Errorf("failed to check/update main SSH config: %w", err))
		}
		if needsInclude {
			// 需要添加Include指令
			if err := addIncludeToSSHConfig(userHomeSSHConfigPath, includeStatement); err != nil {
				panic(fmt.Errorf("failed to add Include statement to main SSH config: %w", err))
			}
		}
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(fmt.Errorf("failed to create docker client: %w", err))
	}
	GlobalConfig.DockerClient = cli
}

// checkAndAddIncludeStatement 检查SSH配置文件是否包含指定的Include语句
func checkAndAddIncludeStatement(configPath, includeStatement string) (bool, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == includeStatement {
			return false, nil // 已存在，不需要添加
		}
	}

	return true, scanner.Err() // 需要添加
}

// addIncludeToSSHConfig 在SSH配置文件的第一行添加Include语句
func addIncludeToSSHConfig(configPath, includeStatement string) error {
	// 读取现有内容
	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	// 在第一行添加Include语句
	newContent := includeStatement + "\n" + string(content)

	// 写回文件
	return os.WriteFile(configPath, []byte(newContent), 0600)
}

func (c *Config) GetDockerClient() *client.Client {
	return c.DockerClient
}
