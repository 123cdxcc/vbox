# VBox - SSH 开发容器

## 构建镜像

```bash
docker build -f ./template/Base-Dockerfile -t devbox ./template
```

## 生成公钥
```
ssh-keygen -t ed25519 -f .ssh/demo -C "demo" -N "" -q
```

## 启动容器

```bash
docker run -d \
  -p 2222:22 \
  -v .ssh/demo.pub:/home/devbox/.ssh/authorized_keys:ro \
  devbox
```

## 连接容器

```bash
ssh -p 2222 devbox@localhost
```