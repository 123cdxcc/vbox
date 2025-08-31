#!/bin/bash

# SSH 开发容器启动脚本
# 用于修复挂载文件的权限并启动 SSH 服务

set -e  # 遇到错误立即退出
set -u  # 使用未定义变量时退出
set -o pipefail  # 管道中任何命令失败都会导致退出

# 颜色输出定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 错误处理函数
error_exit() {
    echo -e "${RED}错误: $1${NC}" >&2
    echo -e "${RED}容器启动失败，正在退出...${NC}" >&2
    exit 1
}

# 成功消息函数
success_msg() {
    echo -e "${GREEN}✓ $1${NC}"
}

# 警告消息函数
warning_msg() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

echo "正在启动 SSH 开发容器..."

# 检查必要的目录是否存在
if [ ! -d "/home/devbox" ]; then
    error_exit "用户目录 /home/devbox 不存在"
fi

if [ ! -d "/home/devbox/.ssh" ]; then
    error_exit "SSH 目录 /home/devbox/.ssh 不存在"
fi

# 检查 authorized_keys 文件
if [ -f /home/devbox/.ssh/authorized_keys ]; then
    echo "检测到 authorized_keys 文件..."
    
    # 检查文件是否可读
    if [ ! -r /home/devbox/.ssh/authorized_keys ]; then
        error_exit "authorized_keys 文件不可读，请检查挂载权限"
    fi
    
    # 检查文件是否为空
    if [ ! -s /home/devbox/.ssh/authorized_keys ]; then
        error_exit "authorized_keys 文件为空，无法进行 SSH 认证"
    fi
    
    # 尝试修复权限（允许失败）
    echo "尝试修复 authorized_keys 权限..."
    if chown devbox:devbox /home/devbox/.ssh/authorized_keys 2>/dev/null; then
        success_msg "文件所有者已修改"
    else
        warning_msg "无法修改文件所有者（可能是只读挂载）"
    fi
    
    if chmod 600 /home/devbox/.ssh/authorized_keys 2>/dev/null; then
        success_msg "文件权限已修改"
    else
        warning_msg "无法修改文件权限（可能是只读挂载）"
    fi
    
    success_msg "authorized_keys 处理完成"
else
    error_exit "未找到 authorized_keys 文件，请确保正确挂载公钥: -v ~/.ssh/demo.pub:/home/devbox/.ssh/authorized_keys:ro"
fi

# 尝试设置 SSH 目录权限（允许失败）
echo "尝试设置 SSH 目录权限..."
if chown -R devbox:devbox /home/devbox/.ssh 2>/dev/null; then
    success_msg "SSH 目录所有者已设置"
else
    warning_msg "无法修改 SSH 目录所有者"
fi

if chmod 700 /home/devbox/.ssh 2>/dev/null; then
    success_msg "SSH 目录权限已设置"
else
    warning_msg "无法修改 SSH 目录权限"
fi

# 检查并生成 SSH 主机密钥
if [ ! -f /etc/ssh/ssh_host_rsa_key ]; then
    echo "生成 SSH 主机密钥..."
    if ! ssh-keygen -A; then
        error_exit "无法生成 SSH 主机密钥"
    fi
    success_msg "SSH 主机密钥已生成"
else
    success_msg "SSH 主机密钥已存在"
fi

# 验证 SSH 配置文件
if [ ! -f /etc/ssh/sshd_config ]; then
    error_exit "SSH 配置文件 /etc/ssh/sshd_config 不存在"
fi

# 测试 SSH 配置是否有效
if ! /usr/sbin/sshd -t; then
    error_exit "SSH 配置文件有语法错误，请检查配置"
fi

success_msg "SSH 配置验证通过"

# 检查工作目录
if [ ! -d "/workspace" ]; then
    warning_msg "工作目录 /workspace 不存在，正在创建..."
    if ! mkdir -p /workspace; then
        error_exit "无法创建工作目录 /workspace"
    fi
    if ! chown devbox:devbox /workspace; then
        error_exit "无法设置工作目录所有者"
    fi
fi

success_msg "工作目录检查完成"

# 显示容器信息
echo -e "\n${GREEN}=== 容器启动成功 ===${NC}"
echo "- 用户: devbox"
echo "- 工作目录: /workspace"
echo "- SSH 端口: 22"
echo "- 配置状态: 所有检查通过"

# 启动 SSH 服务
echo -e "\n启动 SSH 服务..."
if ! exec /usr/sbin/sshd -D; then
    error_exit "SSH 服务启动失败"
fi
