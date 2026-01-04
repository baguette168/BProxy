# 🥖 BProxy 多级级联完整测试指南

## ✅ 已实现的功能

### 1. Agent 级联监听
- ✅ Agent 支持 `-cascade` 参数开启级联端口
- ✅ 自动接受下级 Agent 连接
- ✅ 自动将下级 Agent 注册到 Admin
- ✅ 自动转发下级 Agent 的所有消息

### 2. 拓扑树状显示
- ✅ TUI 显示树状结构（带缩进）
- ✅ 自动识别父子关系
- ✅ 显示 Children 数量
- ✅ 标题改为 🥖 BProxy 🥖

### 3. 消息中继
- ✅ 父 Agent 自动转发子 Agent 消息
- ✅ Admin 自动路由到正确的 Agent
- ✅ 支持多级心跳

---

## 🚀 多级级联测试步骤

### 拓扑结构：Admin → Agent A → Agent B

```
Admin (127.0.0.1:8443)
  ↓
Agent A (级联端口 8444)
  ↓
Agent B (连接到 Agent A:8444)
```

---

## 📋 详细操作步骤

### 步骤 1：从容器复制最新文件

**在宿主机 WSL 执行：**

```bash
# 停止所有正在运行的 BProxy 进程
pkill -f admin-tui
pkill -f agent

# 从容器复制完整项目
docker cp f06323e850af:/workspace/bproxy /home/baguette/openhands_workspace/bproxy_new

# 备份旧版本
mv /home/baguette/openhands_workspace/bproxy /home/baguette/openhands_workspace/bproxy_old

# 使用新版本
mv /home/baguette/openhands_workspace/bproxy_new /home/baguette/openhands_workspace/bproxy

# 进入目录
cd /home/baguette/openhands_workspace/bproxy
```

---

### 步骤 2：启动 Admin（终端 1）

```bash
cd /home/baguette/openhands_workspace/bproxy
./bin/admin-tui -addr 0.0.0.0:8443
```

**预期看到：**
```
🥖 BProxy 🥖

╭───────────────────────────────────────╮╭───────────────────────────────────────╮
│ 📡 Agent Topology (Tree View)         ││ 💻 Console                            │
│                                        ││                                       │
│ No agents connected                    ││ Active Connections: 0                 │
╰───────────────────────────────────────╯╰───────────────────────────────────────╯
```

---

### 步骤 3：启动 Agent A（终端 2）

**关键：使用 `-cascade 8444` 参数！**

```bash
cd /home/baguette/openhands_workspace/bproxy
./bin/agent -admin 127.0.0.1:8443 -cascade 8444
```

**预期日志：**
```
2026/01/04 xx:xx:xx Starting BProxy Agent...
2026/01/04 xx:xx:xx Connecting to admin at 127.0.0.1:8443
2026/01/04 xx:xx:xx Cascade mode enabled on port 8444
2026/01/04 xx:xx:xx Agent <uuid-A> connected to admin
2026/01/04 xx:xx:xx Cascade listener started on port 8444
```

**TUI 应该显示：**
```
📡 Agent Topology (Tree View)

▶ ● <id-A> Baguette [10.255.255.254]
   ↳ Last seen: 1s ago
```

---

### 步骤 4：启动 Agent B（终端 3）

**关键：连接到 Agent A 的级联端口 8444！**

```bash
cd /home/baguette/openhands_workspace/bproxy
./bin/agent -admin 127.0.0.1:8444
```

**预期日志（Agent B）：**
```
2026/01/04 xx:xx:xx Starting BProxy Agent...
2026/01/04 xx:xx:xx Connecting to admin at 127.0.0.1:8444
2026/01/04 xx:xx:xx Agent <uuid-B> connected to admin
```

**预期日志（Agent A）：**
```
2026/01/04 xx:xx:xx Child agent <uuid-B> connected via cascade
2026/01/04 xx:xx:xx Child agent <uuid-B> registered with admin via relay
```

**TUI 应该显示树状结构：**
```
📡 Agent Topology (Tree View)

● <id-A> Baguette [10.255.255.254]
   ↳ Last seen: 1s ago
   ↳ Children: 1

  └─ ● <id-B> Baguette [10.255.255.254]
     ↳ Last seen: 2s ago
```

---

### 步骤 5：验证级联 SOCKS5

**在 TUI 中：**

1. 使用 ↑/↓ 键选择 **Agent B**（子节点）
2. 按 `s` 键启动 SOCKS5

**预期：**
- 控制台显示：`✓ SOCKS5 proxy started on :1080 -> <id-B>`
- SOCKS5 Proxies: 1

**在终端 4 测试：**

```bash
# 检查端口
netstat -tlnp | grep 1080

# 测试代理（流量会通过 Admin → Agent A → Agent B）
curl -x socks5://127.0.0.1:1080 http://httpbin.org/ip
```

**预期输出：**
```json
{
  "origin": "<Agent B 的公网 IP>"
}
```

**日志验证：**
- Admin 日志：`SOCKS5 request: httpbin.org:80 via agent <id-B>`
- Agent A 日志：`Relaying message from child <id-B>`
- Agent B 日志：`Connecting to httpbin.org:80`

---

## 🎯 三级级联测试（可选）

### 拓扑：Admin → Agent A → Agent B → Agent C

```bash
# 终端 1: Admin
./bin/admin-tui -addr 0.0.0.0:8443

# 终端 2: Agent A (级联端口 8444)
./bin/agent -admin 127.0.0.1:8443 -cascade 8444

# 终端 3: Agent B (连接 A，开启级联端口 8445)
./bin/agent -admin 127.0.0.1:8444 -cascade 8445

# 终端 4: Agent C (连接 B)
./bin/agent -admin 127.0.0.1:8445
```

**TUI 应该显示：**
```
📡 Agent Topology (Tree View)

● <id-A> Baguette
   ↳ Children: 1

  └─ ● <id-B> Baguette
     ↳ Children: 1

    └─ ● <id-C> Baguette
       ↳ Last seen: 1s ago
```

---

## 🔍 验证清单

### ✅ 基础功能
- [ ] Admin 启动成功
- [ ] Agent A 连接成功
- [ ] Agent A 显示 "Cascade listener started on port 8444"
- [ ] TUI 显示 Agent A

### ✅ 级联功能
- [ ] Agent B 连接到 Agent A:8444 成功
- [ ] Agent A 日志显示 "Child agent connected via cascade"
- [ ] Agent A 日志显示 "registered with admin via relay"
- [ ] TUI 显示树状结构（Agent B 在 Agent A 下方缩进）
- [ ] TUI 显示 "Children: 1"

### ✅ SOCKS5 级联
- [ ] 选择 Agent B 并按 's' 启动 SOCKS5
- [ ] 端口 1080 监听成功
- [ ] curl 测试返回 Agent B 的 IP
- [ ] 所有三个组件都有相应日志

### ✅ UI 改进
- [ ] 标题显示 "🥖 BProxy 🥖"
- [ ] 拓扑标题显示 "(Tree View)"
- [ ] 子节点有缩进和 "└─" 符号

---

## 🐛 故障排除

### 问题 1：Agent B 无法连接到 Agent A

**检查：**
```bash
# 在 Agent A 的机器上检查端口
netstat -tlnp | grep 8444

# 应该看到
tcp  0  0  0.0.0.0:8444  0.0.0.0:*  LISTEN  <pid>/agent
```

**解决：**
- 确保 Agent A 使用了 `-cascade 8444` 参数
- 检查防火墙是否阻止 8444 端口

### 问题 2：TUI 不显示树状结构

**检查：**
- 确保使用的是最新编译的 `admin-tui`
- 检查文件修改时间：`stat bin/admin-tui`

**解决：**
```bash
# 重新从容器复制
docker cp f06323e850af:/workspace/bproxy/bin/admin-tui ./bin/
```

### 问题 3：Agent B 注册失败

**检查 Agent A 日志：**
- 应该看到 "Child agent <id> connected via cascade"
- 应该看到 "registered with admin via relay"

**如果没有：**
- 检查 Agent A 是否成功连接到 Admin
- 检查 Agent A 的 yamux session 是否正常

---

## 📊 预期的完整日志输出

### Admin 日志
```
2026/01/04 xx:xx:xx Admin server listening on [::]:8443
2026/01/04 xx:xx:xx Agent registered: <id-A> (hostname: Baguette, IPs: [10.255.255.254])
2026/01/04 xx:xx:xx Agent registered: <id-B> (hostname: Baguette, IPs: [10.255.255.254])
2026/01/04 xx:xx:xx Heartbeat from <id-A>
2026/01/04 xx:xx:xx Heartbeat from <id-B>
2026/01/04 xx:xx:xx SOCKS5 proxy started on 127.0.0.1:1080 -> agent <id-B>
2026/01/04 xx:xx:xx SOCKS5 request: httpbin.org:80 via agent <id-B>
```

### Agent A 日志
```
2026/01/04 xx:xx:xx Starting BProxy Agent...
2026/01/04 xx:xx:xx Cascade mode enabled on port 8444
2026/01/04 xx:xx:xx Agent <id-A> connected to admin
2026/01/04 xx:xx:xx Cascade listener started on port 8444
2026/01/04 xx:xx:xx Child agent <id-B> connected via cascade
2026/01/04 xx:xx:xx Child agent <id-B> registered with admin via relay
2026/01/04 xx:xx:xx Relaying message from child <id-B>
```

### Agent B 日志
```
2026/01/04 xx:xx:xx Starting BProxy Agent...
2026/01/04 xx:xx:xx Connecting to admin at 127.0.0.1:8444
2026/01/04 xx:xx:xx Agent <id-B> connected to admin
2026/01/04 xx:xx:xx Connecting to httpbin.org:80
2026/01/04 xx:xx:xx Tunnel established to httpbin.org:80
```

---

## 🎉 成功标志

当您看到以下所有内容时，说明多级级联完全成功：

1. ✅ TUI 标题显示 "🥖 BProxy 🥖"
2. ✅ TUI 显示树状结构，Agent B 在 Agent A 下方缩进
3. ✅ Agent A 日志显示 "Cascade listener started"
4. ✅ Agent A 日志显示 "Child agent registered with admin via relay"
5. ✅ 通过 Agent B 的 SOCKS5 代理能成功访问网络
6. ✅ curl 返回的 IP 是 Agent B 的 IP（不是 Agent A 的）

---

## 📝 命令速查表

```bash
# 复制最新文件
docker cp f06323e850af:/workspace/bproxy /home/baguette/openhands_workspace/bproxy_new

# 启动 Admin
./bin/admin-tui -addr 0.0.0.0:8443

# 启动 Agent A（带级联）
./bin/agent -admin 127.0.0.1:8443 -cascade 8444

# 启动 Agent B（连接 A）
./bin/agent -admin 127.0.0.1:8444

# 检查级联端口
netstat -tlnp | grep 8444

# 测试 SOCKS5
curl -x socks5://127.0.0.1:1080 http://httpbin.org/ip
```

---

**完成时间：** 2026-01-04  
**版本：** 1.2 (多级级联完整版)  
**状态：** ✅ 已实现并可测试