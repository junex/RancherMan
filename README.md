# RancherMan

RancherMan 是一个用 Go 语言开发的 Rancher 集群管理工具,提供了简单直观的图形界面来管理 Rancher 工作负载。

## 功能特性

- 多环境配置管理和切换
- 命名空间和工作负载的可视化管理与搜索
- 工作负载的启动/停止/重新部署,支持批量操作
- Pod状态实时监控和更新
- 端口和访问路径的快速查看
- 数据库密码自动识别和显示
  - MySQL Root密码自动识别
  - MongoDB Root用户名和密码自动识别
- 跳板机配置自动扫描和关联
- 支持工作负载和配置的克隆与导出
  - 支持跨环境和命名空间克隆
  - 支持指定镜像标签进行克隆
  - 支持批量导出为YAML文件
- 支持镜像部署路径的智能追踪
  - 自动关联跳板机上的部署脚本
  - 支持多级目录结构匹配
  - 支持命名空间相关性排序
- 中文界面,操作简单直观

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

1. 首次运行时点击"配置->显示配置"导入配置文件
2. 点击"数据->更新数据"拉取最新的集群信息
3. 点击"更新Pod"获取最新的Pod运行状态
4. 在左侧选择命名空间,可通过搜索框快速定位
5. 在中间列表选择要操作的工作负载(支持多选)
6. 使用右上方按钮进行相应操作:
   - 打开：启动选中的工作负载
   - 关闭：停止选中的工作负载
   - 重新部署：重新部署选中的工作负载
7. 右侧信息区域会显示:
   - 工作负载详细信息
   - Pod运行状态
   - 访问端口和路径
   - 数据库密码(如果是数据库服务)
   - 相关的部署配置信息
8. 克隆和导出功能:
   - 导出configMap: 将当前命名空间的配置导出到YAML文件
   - 克隆configMap: 将配置克隆到其他命名空间
   - 导出workload: 将工作负载导出到YAML文件
   - 克隆workload: 将工作负载克隆到其他命名空间
     - 可选择是否更新镜像标签
     - 支持指定忽略标签更新的工作负载
9. 跳板机配置:
   - 点击"数据->更新跳板机"扫描跳板机配置
   - 自动关联工作负载的部署路径和脚本
   - 支持多级目录结构的智能匹配
   - 根据命名空间相关性进行排序展示

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

### 查看所有打包选项
fyne package --help

### 常用选项：
-name: 指定应用名称
-icon: 指定应用图标
-appID: 指定应用ID（如：com.company.app）
-release: 创建发布版本
-sourceDir: 指定源代码目录


## 许可证

MIT License
