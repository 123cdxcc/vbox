package tools

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/moby/moby/client"
)

// ImageExists 检查镜像是否存在
func ImageExists(ctx context.Context, cli *client.Client, imageName string) (bool, error) {
	_, err := cli.ImageInspect(ctx, imageName)
	if err != nil {
		if strings.Contains(err.Error(), "No such image") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CreateBuildContext 创建构建上下文的 tar 流
func CreateBuildContext(dockerfilePath string, setupScriptPath string, setupEnvScriptPath string) (io.ReadCloser, error) {
	// 创建管道
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		tw := tar.NewWriter(pw)
		defer tw.Close()

		// 添加 Dockerfile 到 tar (在根目录下命名为 Dockerfile)
		if err := addFileToTar(tw, dockerfilePath, filepath.Base(dockerfilePath)); err != nil {
			pw.CloseWithError(fmt.Errorf("添加 Dockerfile 到 tar 失败: %w", err))
			return
		}

		if err := addFileToTar(tw, setupScriptPath, filepath.Base(setupScriptPath)); err != nil {
			pw.CloseWithError(fmt.Errorf("添加 setup.sh 到 tar 失败: %w", err))
			return
		}

		if err := addFileToTar(tw, setupEnvScriptPath, "env.sh"); err != nil {
			pw.CloseWithError(fmt.Errorf("添加环境脚本到 tar 失败: %w", err))
			return
		}
	}()

	return pr, nil
}

// addFileToTar 添加单个文件到 tar
func addFileToTar(tw *tar.Writer, filePath, tarPath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header := &tar.Header{
		Name: tarPath,
		Size: info.Size(),
		Mode: int64(info.Mode()),
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	_, err = io.Copy(tw, file)
	return err
}
