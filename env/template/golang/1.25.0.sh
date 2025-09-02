#!/bin/bash

# Go 环境设置脚本
# 下载并安装 Go 1.25.0

set -e

echo "正在下载 Go 1.25.0..."
wget https://go.dev/dl/go1.25.0.linux-amd64.tar.gz

echo "正在安装 Go..."
rm -rf /usr/local/go
tar -C /usr/local -xzf go1.25.0.linux-amd64.tar.gz
rm go1.25.0.linux-amd64.tar.gz

echo "正在配置 Go 环境变量..."
echo "export PATH=\$PATH:/usr/local/go/bin" >> /home/devbox/.bashrc
echo "export GOPATH=/home/devbox/go" >> /home/devbox/.bashrc
echo "export GOBIN=\$GOPATH/bin" >> /home/devbox/.bashrc

echo "正在创建 Go 工作目录..."
mkdir -p /home/devbox/go/{bin,src,pkg}
chown -R devbox:devbox /home/devbox/go

echo "Go 1.25.0 安装完成！"
