package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/123cdxcc/vbox/constant"
)

// GetTemplatePath 根据镜像名称获取对应的模板路径
func GetTemplatePath(templatesDirPath, imageName string) (string, error) {
	// 从镜像名称提取模板名称
	// 例如：vbox-base -> base, vbox-dev -> dev
	parts := strings.Split(imageName, "-")
	if len(parts) < 2 || parts[0] != constant.VboxCommonPrefix {
		return "", fmt.Errorf("镜像名称格式错误，应为 vbox-xxx 格式")
	}

	templateName := strings.Join(parts[1:], "-") // 支持多段名称，如 vbox-web-server
	templatePath := filepath.Join(templatesDirPath, templateName)
	// 检查模板文件是否存在（忽略大小写）
	entries, err := os.ReadDir(templatesDirPath)
	if err != nil {
		return "", fmt.Errorf("无法读取模板目录 %s: %w", templatesDirPath, err)
	}

	flag := false
	// 将目标文件名转为小写进行匹配
	targetFileName := strings.ToLower(templateName)
	for _, entry := range entries {
		if !entry.IsDir() && strings.ToLower(entry.Name()) == targetFileName {
			// 找到匹配的文件，使用实际的文件名
			templatePath = filepath.Join(templatesDirPath, entry.Name())
			flag = true
			break
		}
	}

	if !flag {
		return "", fmt.Errorf("模板文件 %s 不存在", templatePath)
	}

	return templatePath, nil
}
