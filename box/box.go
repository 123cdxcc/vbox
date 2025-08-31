package box

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/123cdxcc/vbox/config"
	"github.com/123cdxcc/vbox/constant"
	"github.com/123cdxcc/vbox/pkg/tools"
	"github.com/docker/go-connections/nat"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

// 自定义错误
var ErrBoxNotFound = errors.New("box not found")

type PortType string

const (
	PortTypeTCP PortType = "tcp"
	PortTypeUDP PortType = "udp"
)

type Port struct {
	IP          string   `json:"ip,omitempty"`
	PrivatePort int      `json:"private_port"`
	PublicPort  int      `json:"public_port,omitempty"`
	Type        PortType `json:"type"`
}

// convertPorts 将 container.Port 转换为我们的 Port 类型
func convertPorts(containerPorts []container.Port) []Port {
	ports := make([]Port, len(containerPorts))
	for i, port := range containerPorts {
		ports[i] = Port{
			IP:          port.IP,
			PrivatePort: int(port.PrivatePort),
			PublicPort:  int(port.PublicPort),
			Type:        PortType(port.Type),
		}
	}
	return ports
}

func convertPortMapToPorts(portMap container.PortMap) []Port {
	ports := make([]Port, 0, len(portMap))
	for k, v := range portMap {
		if len(v) > 0 {
			// 解析端口类型和端口号
			portStr := string(k)
			var protocol string
			var containerPort int

			// 分离端口号和协议
			if strings.Contains(portStr, "/") {
				parts := strings.Split(portStr, "/")
				if len(parts) == 2 {
					if port, err := strconv.Atoi(parts[0]); err == nil {
						containerPort = port
						protocol = parts[1]
					}
				}
			}

			for _, host := range v {
				if hostPort, err := strconv.Atoi(host.HostPort); err == nil {
					ports = append(ports, Port{
						IP:          host.HostIP,
						PrivatePort: containerPort,
						PublicPort:  hostPort,
						Type:        PortType(protocol),
					})
				}
			}
		}
	}
	return ports
}

func convertContainer(box container.Summary) (*Container, bool) {
	for _, name := range box.Names {
		if strings.HasPrefix(tools.EscapeDockerName(name), constant.VboxContainerPrefix) {
			return &Container{
				ID:     box.ID,
				Name:   strings.TrimPrefix(tools.EscapeDockerName(name), constant.VboxContainerPrefix),
				Image:  strings.TrimPrefix(tools.EscapeDockerName(box.Image), constant.VboxContainerPrefix),
				Status: box.Status,
				State:  box.State,
				Ports:  convertPorts(box.Ports),
			}, true
		}
	}
	return nil, false
}

// Container 表示一个Docker容器的信息
type Container struct {
	ID     string
	Name   string
	Image  string
	Status string
	State  string
	Ports  []Port
}

// List 列出所有以"vbox-"开头的Docker容器
// 返回容器列表，包括运行中和已停止的容器
func List() ([]Container, error) {
	cli := config.GlobalConfig.GetDockerClient()

	// 列出所有容器（包括已停止的）
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{
		All: true, // 包括已停止的容器
	})
	if err != nil {
		return nil, err
	}

	var vboxContainers []Container
	for _, box := range containers {
		vboxContainer, ok := convertContainer(box)
		if ok {
			vboxContainers = append(vboxContainers, *vboxContainer)
		}
	}

	return vboxContainers, nil
}

// Get 根据容器ID获取单个以"vbox-"开头的容器
// 如果容器不存在或不是vbox容器，返回ErrBoxNotFound错误
func Get(ctx context.Context, containerID string) (*Container, error) {
	cli := config.GlobalConfig.GetDockerClient()
	boxInfo, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, err
	}

	result := &Container{
		ID:     boxInfo.ID,
		Name:   strings.TrimPrefix(tools.EscapeDockerName(boxInfo.Name), constant.VboxContainerPrefix),
		Image:  boxInfo.Config.Image,
		Status: boxInfo.State.Status,
		State:  boxInfo.State.Status,
		Ports:  convertPortMapToPorts(boxInfo.NetworkSettings.Ports),
	}
	return result, nil
}

type CreateOption struct {
	Name         string
	ImageName    string
	ImageVersion string
	Ports        []Port
	SSHPort      int               // SSH 端口映射，默认 2222
	PublicKey    string            // SSH 公钥内容或文件路径
	Volumes      map[string]string // 卷挂载配置，key为主机路径，value为容器路径
	Detached     bool              // 是否后台运行，默认 true
}

// ensureVboxNetwork 确保 vbox 专用网络存在
func ensureVboxNetwork(ctx context.Context, cli *client.Client) error {
	const networkName = constant.VboxNetwork

	// 检查网络是否已存在
	networks, err := cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return fmt.Errorf("列出网络失败: %w", err)
	}

	for _, net := range networks {
		if net.Name == networkName {
			return nil // 网络已存在
		}
	}

	// 创建 vbox 专用网络
	_, err = cli.NetworkCreate(ctx, networkName, network.CreateOptions{
		Driver: constant.DefaultNetworkDriver,
		IPAM: &network.IPAM{
			Config: []network.IPAMConfig{
				{
					Subnet: constant.VboxNetworkSubnet,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("创建 vbox 网络失败: %w", err)
	}

	return nil
}

func Create(ctx context.Context, opt CreateOption) (*Container, error) {
	cli := config.GlobalConfig.GetDockerClient()
	image := fmt.Sprintf("%s%s:%s", constant.VboxImagePrefix, opt.ImageName, opt.ImageVersion)
	opt.Name = constant.VboxContainerPrefix + opt.Name
	// 确保 vbox 网络存在
	if err := ensureVboxNetwork(ctx, cli); err != nil {
		return nil, err
	}

	// 检查容器是否已存在
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("列出容器失败: %w", err)
	}

	for _, c := range containers {
		for _, name := range c.Names {
			cleanName := strings.TrimPrefix(name, "/")
			if cleanName == opt.Name {
				return nil, fmt.Errorf("容器 %s 已存在", opt.Name)
			}
		}
	}

	// 创建容器配置
	boxConfig := &container.Config{
		Image: image,
	}

	// 配置端口
	allPorts := opt.Ports

	// 如果指定了 SSH 端口，添加 SSH 端口映射
	if opt.SSHPort > 0 {
		sshPort := Port{
			PrivatePort: 22,
			PublicPort:  opt.SSHPort,
			Type:        PortTypeTCP,
		}
		allPorts = append(allPorts, sshPort)
	}

	// 如果有端口配置，添加到容器配置中
	if len(allPorts) > 0 {
		boxConfig.ExposedPorts = make(nat.PortSet)
		for _, port := range allPorts {
			portKey := nat.Port(fmt.Sprintf("%d/%s", port.PrivatePort, port.Type))
			boxConfig.ExposedPorts[portKey] = struct{}{}
		}
	}

	hostConfig := &container.HostConfig{
		NetworkMode: constant.VboxNetwork, // 连接到 vbox 专用网络
	}

	// 配置端口映射
	if len(allPorts) > 0 {
		hostConfig.PortBindings = make(nat.PortMap)
		for _, port := range allPorts {
			portKey := nat.Port(fmt.Sprintf("%d/%s", port.PrivatePort, port.Type))
			if port.PublicPort > 0 {
				hostConfig.PortBindings[portKey] = []nat.PortBinding{
					{
						HostIP:   "0.0.0.0",
						HostPort: fmt.Sprintf("%d", port.PublicPort),
					},
				}
			}
		}
	}

	// 配置卷挂载
	if len(opt.Volumes) > 0 {
		binds := make([]string, 0, len(opt.Volumes))
		for hostPath, containerPath := range opt.Volumes {
			bind := fmt.Sprintf("%s:%s", hostPath, containerPath)
			binds = append(binds, bind)
		}
		hostConfig.Binds = binds
	}

	// 如果指定了公钥，添加公钥挂载
	if opt.PublicKey != "" {
		// 检查是否是文件路径
		if stat, err := os.Stat(opt.PublicKey); err == nil {
			// 确保是文件而不是目录
			if stat.IsDir() {
				return nil, fmt.Errorf("公钥路径 %s 是一个目录，不是文件", opt.PublicKey)
			}
			// 是文件路径，挂载为只读
			keyBind := fmt.Sprintf("%s:%s:ro", opt.PublicKey, constant.DefaultSSHAuthorizedKeysPath)
			if hostConfig.Binds == nil {
				hostConfig.Binds = []string{}
			}
			hostConfig.Binds = append(hostConfig.Binds, keyBind)
		} else {
			return nil, fmt.Errorf("公钥文件 %s 不存在或无法访问: %w", opt.PublicKey, err)
		}
		// TODO: 非本地文件
	}

	// 创建容器
	resp, err := cli.ContainerCreate(ctx, boxConfig, hostConfig, nil, nil, opt.Name)
	if err != nil {
		return nil, fmt.Errorf("创建容器失败: %w", err)
	}

	// 启动容器
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("启动容器失败: %w", err)
	}

	// 获取容器信息
	containerInfo, err := cli.ContainerInspect(ctx, resp.ID)
	if err != nil {
		return nil, fmt.Errorf("获取容器信息失败: %w", err)
	}
	// 构造返回的 Container 结构体
	result := &Container{
		ID:     containerInfo.ID,
		Name:   strings.TrimPrefix(opt.Name, constant.VboxContainerPrefix),
		Image:  containerInfo.Config.Image,
		Status: containerInfo.State.Status,
		State:  containerInfo.State.Status,
		Ports:  convertPortMapToPorts(containerInfo.NetworkSettings.Ports),
	}

	return result, nil
}

// Stop 停止指定的容器
// containerID 必须是容器ID
func Stop(ctx context.Context, containerID string) error {
	cli := config.GlobalConfig.GetDockerClient()

	// 停止容器
	err := cli.ContainerStop(ctx, containerID, container.StopOptions{})
	if err != nil {
		return fmt.Errorf("停止容器失败: %w", err)
	}

	return nil
}

// Delete 删除指定的容器
// containerID 必须是容器ID
// force 参数决定是否强制删除运行中的容器
func Delete(ctx context.Context, containerID string, force bool) error {
	cli := config.GlobalConfig.GetDockerClient()

	// 删除容器
	err := cli.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force: force, // 是否强制删除运行中的容器
	})
	if err != nil {
		return fmt.Errorf("删除容器失败: %w", err)
	}

	return nil
}

// StopAndDelete 停止并删除指定的容器
// containerID 必须是容器ID
// 这是一个便捷方法，先停止容器然后删除
func StopAndDelete(ctx context.Context, containerID string) error {
	// 先停止容器
	if err := Stop(ctx, containerID); err != nil {
		// 如果容器已经停止，继续删除流程
		if !strings.Contains(err.Error(), "is not running") {
			return err
		}
	}

	// 删除容器
	return Delete(ctx, containerID, false)
}
