# GitBook CLI (Go 重写版)

一个基于 Go 语言重写的 GitBook 命令行工具，提供简洁高效的 GitBook 常用操作。

## 项目背景

GitBook CLI 是一个已被废弃的 JavaScript 项目。虽然社区有贡献者基于 TypeScript 进行了重写，但使用体验仍不够理想。本项目使用 Go 语言和简单的前端技术，对 GitBook 的常用操作进行了重新实现，旨在提供更稳定、更快速的工具体验。

## 特性

- 🚀 **高性能**: 基于 Go 语言，启动速度快，资源占用低
- 📦 **轻量级**: 单一二进制文件，无需 Node.js 环境
- 🔧 **简单易用**: 保持与原生 GitBook CLI 相似的命令接口
- 📚 **功能完整**: 支持书籍初始化、本地预览、静态构建、电子书导出等核心功能
- 🎨 **现代化界面**: 简洁美观的前端预览界面

## 安装

### 从源码构建

```bash
git clone https://github.com/hitzhangjie/gitbook.git
cd gitbook/gitbook
go build -o gitbook
```

### 使用 Go 安装

```bash
go install github.com/hitzhangjie/gitbook@latest
```

## 快速开始

### 初始化一个 GitBook 项目

```bash
gitbook init
```

这将在当前目录创建一个新的 GitBook 项目，包含：
- `book.json` - 书籍配置文件
- `README.md` - 书籍介绍
- `SUMMARY.md` - 目录结构

### 本地预览

```bash
gitbook serve
```

启动本地开发服务器，默认在 `http://localhost:4000` 预览你的书籍。支持热重载，修改文件后自动刷新。

#### 阅读设置

点击左侧目录栏顶部的设置按钮可以打开阅读设置面板：

| 设置 | 说明 |
|------|------|
| 左侧章节目录宽度 | 调整左侧章节目录栏的宽度 |
| 右侧本章目录宽度 | 调整右侧当前章节目录栏的宽度 |
| 记住每章的阅读位置 | 默认开启。开启时重新打开读过的章节会回到上次停下的位置；关闭后每次都从章节开头显示 |

这些选择保存在本机浏览器的 `localStorage` 中，下次打开自动恢复；点击「重置设置」可恢复默认值。

阅读位置本身按章节记录在浏览器当前标签页的 `sessionStorage` 中：同一标签页内切换章节、刷新页面都会保留，关闭标签页后清空，服务端不会记录任何阅读进度。关闭「记住每章的阅读位置」时会同时清除已记录的阅读位置。

以上阅读界面和设置同样适用于 `gitbook build` 生成的静态站点。

### 构建静态网站

```bash
gitbook build [book] [output]
```

将 GitBook 项目构建为静态网站，输出到指定目录。

### 导出电子书

支持导出为多种格式：

```bash
# 导出 PDF
gitbook pdf [book] [output]

# 导出 EPUB
gitbook epub [book] [output]

# 导出 MOBI
gitbook mobi [book] [output]
```

## 命令说明

| 命令 | 说明 | 用法 |
|------|------|------|
| `init` | 初始化一个新的 GitBook 项目 | `gitbook init [directory]` |
| `serve` | 启动本地预览服务器 | `gitbook serve [book]` |
| `build` | 构建静态网站 | `gitbook build [book] [output]` |
| `pdf` | 导出为 PDF 格式 | `gitbook pdf [book] [output]` |
| `epub` | 导出为 EPUB 格式 | `gitbook epub [book] [output]` |
| `mobi` | 导出为 MOBI 格式 | `gitbook mobi [book] [output]` |
| `version` | 显示版本信息 | `gitbook version` |

## 项目结构

```
gitbook/
├── gitbook/              # 主程序
│   ├── main.go          # 入口文件
│   ├── commands/        # 命令实现
│   │   ├── cmd_init.go
│   │   ├── cmd_serve.go
│   │   ├── cmd_build.go
│   │   ├── cmd_pdf.go
│   │   ├── cmd_epub.go
│   │   ├── cmd_mobi.go
│   │   └── cmd_version.go
│   ├── book/            # 书籍配置和解析
│   ├── builder/          # 静态网站构建器
│   ├── server/           # 本地预览服务器
│   └── ebook/            # 电子书生成器
└── README.md
```

## 技术栈

- **后端**: Go 1.24+
- **前端**: 原生 JavaScript + CSS
- **依赖管理**: Go Modules
- **命令行框架**: Cobra

## 开发

### 环境要求

- Go 1.24 或更高版本

### 构建

```bash
cd gitbook
go build -o gitbook
```

### 运行测试

```bash
go test ./...
```

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

本项目采用 MIT 许可证。

## 致谢

本项目基于已废弃的 GitBook CLI 2.3.2 版本重写，感谢原项目及其社区贡献者的工作。

