🥖 BProxy - 多级 SOCKS5 代理工具

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/Platform-Windows%20%7C%20Linux-lightgrey" alt="Platform">
  <img src="https://img.shields.io/badge/License-MIT-green" alt="License">
</p>

BProxy 是一款面向渗透测试场景的多级 SOCKS5 代理工具，专为解决复杂内网环境（如 Active Directory 域）中的代理需求而设计。

## 🎯 开发背景

在 OSCP 考试和日常渗透测试练习中，面对 AD 域环境时，传统工具（如 Chisel、Ligolo-ng、proxychains 等）存在以下痛点：

- 配置复杂，需要记忆大量参数
- 多级代理搭建繁琐
- 缺乏直观的拓扑可视化
- 工具切换频繁，效率低下

BProxy 旨在提供一个**统一、简洁、可视化**的多级代理解决方案。

## ✨ 核心特性

- **🔗 多级级联代理**：支持无限级 Agent 级联（Admin → Agent1 → Agent2 → Agent3 → ... ）
- **🖥️ TUI 管理界面**：实时查看网络拓扑树，一键启动 SOCKS5 代理
- **🔒 TLS 加密通信**：所有通信流量使用 TLS 加密
- **💓 心跳保活机制**：自动检测节点存活状态
- **🚀 跨平台支持**：支持 Windows/Linux，使用 Go 语言编写

## 📐 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                        Admin (TUI)                          │
│                    监听端口:  8443                           │
│              SOCKS5 代理端口: 1080                           │
└─────────────────┬───────────────────────────────────────────┘
                  │ TLS
                  ▼
┌─────────────────────────────────────────────────────────────┐
│                     Agent1 (一级代理)                        │
│                连接 Admin: 8443 + 级联端口 9443              │
└─────────────────┬───────────────────────────────────────────┘
                  │ TLS
                  ▼
┌─────────────────────────────────────────────────────────────┐
│                     Agent2 (二级代理)                        │
│               连接 Agent1:9443 + 级联端口 9444               │
└─────────────────┬───────────────────────────────────────────┘
                  │ TLS
                  ▼
┌─────────────────────────────────────────────────────────────┐
│                     Agent3 (三级代理)                        │
│                    连接 Agent2:9444                          │
│                   可访问目标内网资源                          │
└─────────────────────────────────────────────────────────────┘
```

## 🚀 快速开始

### 编译

```bash
cd bproxy

# 编译 Admin TUI
go build -o bin/admin-tui cmd/admin-tui/main. go

# 编译 Agent (Linux)
go build -o bin/agent cmd/agent/main.go

# 编译 Agent (Windows)
GOOS=windows GOARCH=amd64 go build -o bin/agent.exe cmd/agent/main.go
```

### 使用示例：三级代理

**场景**：攻击机 → 边界服务器 → 内网服务器 → AD 域控制器

#### 1. 在攻击机启动 Admin

```bash
./admin-tui -addr 0.0.0.0:8443
```

#### 2. 在边界服务器启动 Agent1

```bash
./agent -admin <攻击机IP>:8443 -cascade 9443
```

#### 3. 在内网服务器启动 Agent2

```bash
./agent -admin <边界服务器IP>:9443 -cascade 9444
```

#### 4. 在可访问域控的主机启动 Agent3

```bash
./agent -admin <内网服务器IP>:9444
```

#### 5. 启动 SOCKS5 代理

在 Admin TUI 中：

- 使用方向键选择 Agent3
- 按 `s` 启动 SOCKS5 代理（默认端口 1080）

#### 6. 使用代理访问内网资源

```bash
# 使用 curl 测试
curl --socks5 127.0.0.1:1080 http://内网目标IP

# 配合 proxychains
proxychains nmap -sT -Pn 内网目标IP

# 使用 Evil-WinRM 连接域控
proxychains evil-winrm -i DC_IP -u Administrator -p Password
```

## 🎮 TUI 操作说明

| 按键  | 功能  |
| --- | --- |
| `↑/↓` | 选择 Agent |
| `s` | 启动 SOCKS5 代理 |
| `x` | 停止 SOCKS5 代理 |
| `q` | 退出程序 |

## 📁 项目结构

```
bproxy/
├── cmd/
│   ├── admin-tui/     # Admin TUI 入口
│   └── agent/         # Agent 入口
├── admin/             # Admin 核心逻辑
├── agent/             # Agent 核心逻辑
├── pkg/
│   ├── protocol/      # 消息协议处理
│   ├── socks5/        # SOCKS5 实现
│   ├── tls/           # TLS 配置
│   └── topology/      # 拓扑管理
├── proto/             # Protobuf 定义
└── bin/               # 编译输出
```

## 🛣️ 路线图

- [x] 多级 SOCKS5 代理
- [x] TUI 拓扑可视化
- [x] TLS 加密通信
- [ ] HTTP 代理支持
- [ ] 端口转发功能
- [ ] 文件传输功能
- [ ] 命令执行功能
- [ ] Web 管理界面
- [ ] 流量混淆（DNS/ICMP 隧道）
待实现
