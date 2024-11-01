# RancherMan

RancherMan 是一个用 Go 语言开发的 Rancher 集群管理工具,提供了简单直观的图形界面来管理 Rancher 工作负载。

## 功能特性

- 支持多环境配置管理
- 命名空间和工作负载的可视化管理
- 工作负载的启动/停止/重新部署
- 支持工作负载的批量操作
- 端口和访问路径的快速查看
- 支持中文界面

## 安装

1. 下载最新的发布版本
2. 解压后直接运行可执行文件

## 配置说明

配置文件采用 YAML 格式,示例如下:
```yaml
environment:
    dev: # 环境标识
        name: "开发环境" # 环境名称
        base_url: "xxx" # Rancher API地址
        project: "xxx" # 项目ID
        ip: "xxx.xxx.xxx.xxx" # 环境IP
        key: # API密钥
            name: "xxx"
            token: "xxx"
        nginx: # Nginx配置(可选)
            main:
                base_url: "xxx"
                nginx_conf: "xxx"
```

## 使用说明

1. 首次运行时点击"加载配置"按钮导入配置文件
2. 点击"更新数据"拉取最新的集群信息
3. 在左侧选择命名空间
4. 在中间列表选择要操作的工作负载
5. 使用右上方按钮进行相应操作

## 开发说明

本项目使用以下主要依赖:

- [Fyne](https://fyne.io/) - GUI框架
- [GORM](https://gorm.io/) - ORM框架
- SQLite - 本地数据存储

## 构建

1. 确保已安装 fyne 命令行工具：
```bash
go install fyne.io/fyne/v2/cmd/fyne@latest
```

```bash
# windows
fyne package -os windows -name Rancher助手
# macos
fyne package -os darwin -name Rancher助手
# linux
fyne package -os linux -name Rancher助手
```

### 跨平台打包的注意事项：
1. 首先需要安装 fyne-cross：
```bash
go install github.com/fyne-io/fyne-cross@latest
```

2. 使用 fyne-cross 进行跨平台打包：
```bash
# Windows 64位
fyne-cross windows -arch=amd64
# MacOS
fyne-cross darwin -arch=amd64
# Linux
fyne-cross linux -arch=amd64
```

# 查看所有打包选项
fyne package --help

# 常用选项：
-name: 指定应用名称
-icon: 指定应用图标
-appID: 指定应用ID（如：com.company.app）
-release: 创建发布版本
-sourceDir: 指定源代码目录


## 许可证

MIT License
