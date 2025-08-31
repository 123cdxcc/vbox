package config

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

// SSHHostConfig SSH主机配置结构体
type SSHHostConfig struct {
	Name                  string
	HostName              string
	Port                  string
	User                  string
	IdentityFile          string
	IdentitiesOnly        bool
	StrictHostKeyChecking bool
}

// String 格式化SSH配置为字符串
func (c *SSHHostConfig) String() string {
	identitiesOnly := "yes"
	if !c.IdentitiesOnly {
		identitiesOnly = "no"
	}
	strictHostKeyChecking := "no"
	if c.StrictHostKeyChecking {
		strictHostKeyChecking = "yes"
	}

	return fmt.Sprintf(`Host %s
    HostName %s
    Port %s
    User %s
    IdentityFile %s
    IdentitiesOnly %s
    StrictHostKeyChecking %s`,
		c.Name, c.HostName, c.Port, c.User, c.IdentityFile, identitiesOnly, strictHostKeyChecking)
}

func UpdateSSH(name, host, user, port, privateKey, publicKey string) (string, error) {
	configPath := GlobalConfig.AppSSHConfigPath
	privateKeyPath := filepath.Join(GlobalConfig.AppSSHDirPath, name)
	publicKeyPath := filepath.Join(GlobalConfig.AppSSHDirPath, name+".pub")

	// 确保SSH配置目录存在
	if err := os.MkdirAll(GlobalConfig.AppSSHDirPath, 0700); err != nil {
		return "", fmt.Errorf("failed to create SSH config directory: %w", err)
	}

	// 保存私钥到文件
	if err := os.WriteFile(privateKeyPath, []byte(privateKey), 0600); err != nil {
		return "", fmt.Errorf("failed to write private key: %w", err)
	}

	// 保存公钥到文件
	if err := os.WriteFile(publicKeyPath, []byte(publicKey), 0600); err != nil {
		return "", fmt.Errorf("failed to write public key: %w", err)
	}

	// 创建新的SSH配置
	newConfig := &SSHHostConfig{
		Name:                  name,
		HostName:              host,
		Port:                  port,
		User:                  user,
		IdentityFile:          privateKeyPath,
		IdentitiesOnly:        true,
		StrictHostKeyChecking: false,
	}

	// 读取现有配置
	configs, err := readSSHConfigs(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read SSH config: %w", err)
	}

	// 检查是否存在同名配置，如果存在则更新，否则添加
	updated := false
	for i, config := range configs {
		if config.Name == name {
			configs[i] = newConfig
			updated = true
			break
		}
	}
	if !updated {
		configs = append(configs, newConfig)
	}

	// 写入配置文件
	err = writeSSHConfigs(configPath, configs)
	if err != nil {
		return "", fmt.Errorf("failed to write SSH config: %w", err)
	}

	return publicKeyPath, nil
}

// RemoveSSH 删除SSH配置
func RemoveSSH(name string) error {
	configPath := GlobalConfig.AppSSHConfigPath

	// 读取现有配置
	configs, err := readSSHConfigs(configPath)
	if err != nil {
		return fmt.Errorf("failed to read SSH config: %w", err)
	}

	// 查找要删除的配置
	var targetConfig *SSHHostConfig
	var remainingConfigs []*SSHHostConfig

	for _, config := range configs {
		if config.Name == name {
			targetConfig = config
		} else {
			remainingConfigs = append(remainingConfigs, config)
		}
	}

	// 如果没有找到要删除的配置
	if targetConfig == nil {
		return fmt.Errorf("SSH host '%s' not found", name)
	}

	// 检查是否需要删除IdentityFile
	shouldDeleteIdentityFile := true
	for _, config := range remainingConfigs {
		if config.IdentityFile == targetConfig.IdentityFile {
			shouldDeleteIdentityFile = false
			break
		}
	}

	// 删除IdentityFile（如果没有其他配置使用）
	if shouldDeleteIdentityFile {
		if err := os.Remove(targetConfig.IdentityFile); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove identity file: %w", err)
		}
		// 删除对应的公钥文件
		publicKeyPath := targetConfig.IdentityFile + ".pub"
		if err := os.Remove(publicKeyPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove public key file: %w", err)
		}
	}

	// 写入更新后的配置文件
	return writeSSHConfigs(configPath, remainingConfigs)
}

// GenSSHKeys 生成ssh所需证书
// Returns publicKey, privateKey, error
func GenSSHKeys() (string, string, error) {
	// 生成RSA私钥
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// 编码私钥为PEM格式
	privateKeyDER := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyDER,
	})

	// 生成公钥
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate public key: %w", err)
	}

	// 格式化公钥为OpenSSH格式
	publicKeyBytes := ssh.MarshalAuthorizedKey(publicKey)

	return string(publicKeyBytes), string(privateKeyPEM), nil
}

// readSSHConfigs 读取SSH配置文件
func readSSHConfigs(configPath string) ([]*SSHHostConfig, error) {
	var configs []*SSHHostConfig

	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return configs, nil // 文件不存在，返回空配置
		}
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentConfig *SSHHostConfig

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "Host ") {
			// 保存之前的配置
			if currentConfig != nil {
				configs = append(configs, currentConfig)
			}
			// 创建新配置
			hostName := strings.TrimSpace(strings.TrimPrefix(line, "Host"))
			currentConfig = &SSHHostConfig{
				Name:                  hostName,
				IdentitiesOnly:        true,
				StrictHostKeyChecking: false,
			}
		} else if currentConfig != nil {
			// 解析配置项
			parts := strings.SplitN(line, " ", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch key {
			case "HostName":
				currentConfig.HostName = value
			case "Port":
				currentConfig.Port = value
			case "User":
				currentConfig.User = value
			case "IdentityFile":
				currentConfig.IdentityFile = value
			case "IdentitiesOnly":
				currentConfig.IdentitiesOnly = value == "yes"
			case "StrictHostKeyChecking":
				currentConfig.StrictHostKeyChecking = value == "yes"
			}
		}
	}

	// 添加最后一个配置
	if currentConfig != nil {
		configs = append(configs, currentConfig)
	}

	return configs, scanner.Err()
}

// writeSSHConfigs 写入SSH配置文件
func writeSSHConfigs(configPath string, configs []*SSHHostConfig) error {
	file, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	for i, config := range configs {
		if i > 0 {
			file.WriteString("\n")
		}
		file.WriteString(config.String())
		file.WriteString("\n")
	}

	return nil
}
