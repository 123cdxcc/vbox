# VBox 开发容器

## 构建镜像

```bash
vbox image build -n golang -v 1.25.0 
```

## 启动容器

```bash
vbox run --name golang-demo golang:1.25.0
```

## 连接容器

```bash
ssh golang-demo
```