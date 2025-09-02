package template

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
)

//go:embed template
var envFS embed.FS

//go:embed Dockerfile
var baseDockerfile []byte

//go:embed setup.sh
var setupScript []byte

type Env struct {
	Name    string
	Version string
	Data    []byte
}

var envs []Env

func init() {
	files, err := envFS.ReadDir("template")
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		versionFS, err := envFS.ReadDir(filepath.Join("template", file.Name()))
		if err != nil {
			panic(err)
		}
		for _, versionFile := range versionFS {
			env := Env{
				Name: file.Name(),
			}
			env.Version = strings.TrimSuffix(versionFile.Name(), ".sh")
			env.Data, err = envFS.ReadFile(filepath.Join("template", file.Name(), versionFile.Name()))
			if err != nil {
				panic(err)
			}
			envs = append(envs, env)
		}
	}
}

func Init(targetDir string) error {
	// 创建目标目录
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}

	// 写入 Dockerfile
	dockerfilePath := filepath.Join(targetDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, baseDockerfile, 0644); err != nil {
		return err
	}

	// 写入 setup.sh
	setupScriptPath := filepath.Join(targetDir, "setup.sh")
	if err := os.WriteFile(setupScriptPath, setupScript, 0644); err != nil {
		return err
	}

	// 创建 template 子目录
	templateDir := filepath.Join(targetDir, "template")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		return err
	}

	// 遍历 envs 并创建目录结构和文件
	for _, env := range envs {
		// 创建环境目录 (如 golang)
		envDir := filepath.Join(templateDir, env.Name)
		if err := os.MkdirAll(envDir, 0755); err != nil {
			return err
		}

		// 写入版本文件
		versionFilePath := filepath.Join(envDir, env.Version+".sh")
		if err := os.WriteFile(versionFilePath, env.Data, 0644); err != nil {
			return err
		}
	}

	return nil
}
