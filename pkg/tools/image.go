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
func CreateBuildContext(dockerfilePath string) (io.ReadCloser, error) {
	// 获取 Dockerfile 所在的目录作为构建上下文
	contextDir := filepath.Dir(dockerfilePath)

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

		// 递归添加 Dockerfile 所在目录中的所有其他文件，忽略 Dockerfile 文件
		if err := addDirToTar(tw, contextDir, "", dockerfilePath); err != nil {
			pw.CloseWithError(fmt.Errorf("添加上下文目录到 tar 失败: %w", err))
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

// addDirToTar 递归添加目录到 tar
func addDirToTar(tw *tar.Writer, srcDir, baseInTar string, ignoreFiles ...string) error {
	// 创建忽略文件的 map 以便快速查找
	ignoreMap := make(map[string]bool)
	for _, ignoreFile := range ignoreFiles {
		absIgnoreFile, err := filepath.Abs(ignoreFile)
		if err == nil {
			ignoreMap[absIgnoreFile] = true
		}
	}

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录本身
		if info.IsDir() {
			return nil
		}

		// 检查是否需要忽略此文件
		absPath, err := filepath.Abs(path)
		if err == nil && ignoreMap[absPath] {
			return nil
		}

		// 计算在 tar 中的相对路径
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		tarPath := filepath.Join(baseInTar, relPath)
		// 确保使用 Unix 风格的路径分隔符
		tarPath = strings.ReplaceAll(tarPath, "\\", "/")

		return addFileToTar(tw, path, tarPath)
	})
}
