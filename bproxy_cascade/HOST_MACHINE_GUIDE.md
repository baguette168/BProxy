# 🖥️ 宿主机测试指南

## ❗ 重要说明

`/workspace/bproxy` 是 **Docker 容器内部**的路径，不是您宿主机的路径！

## 📁 如何找到项目文件

### 方法 1：检查 Docker 映射目录

您提到有映射目录 `/opt/workspace_base/`，请在宿主机执行：

```bash
# 在宿主机 WSL 中执行
ls -la /opt/workspace_base/

# 如果看到 bproxy 目录
cd /opt/workspace_base/bproxy
ls -la
```

### 方法 2：从 Docker 容器复制文件

如果映射目录中没有文件，您需要从容器复制：

```bash
# 1. 查找容器 ID
docker ps

# 2. 从容器复制文件（替换 <container-id>）
docker cp <container-id>:/workspace/bproxy /opt/workspace_base/

# 3. 进入目录
cd /opt/workspace_base/bproxy
```

### 方法 3：直接在容器内测试

```bash
# 进入容器
docker exec -it <container-id> /bin/bash

# 在容器内执行
cd /workspace/bproxy
./bin/admin-tui -addr 0.0.0.0:8443
```

## 🎯 完整测试步骤（宿主机）

假设文件在 `/opt/workspace_base/bproxy`：

### 终端 1：启动 Admin

```bash
cd /opt/workspace_base/bproxy
./bin/admin-tui -addr 0.0.0.0:8443
```

### 终端 2：启动 Agent

```bash
cd /opt/workspace_base/bproxy
./bin/agent -admin 127.0.0.1:8443
```

### 终端 3：验证和测试

```bash
# 检查端口
netstat -tlnp | grep 1080

# 测试 SOCKS5
curl -x socks5://127.0.0.1:1080 http://httpbin.org/ip
```

## 🔍 查找项目的命令

```bash
# 在宿主机执行，查找 bproxy 目录
find /opt -name "bproxy" -type d 2>/dev/null
find /home -name "bproxy" -type d 2>/dev/null
find /mnt -name "bproxy" -type d 2>/dev/null

# 查找 admin-tui 二进制文件
find / -name "admin-tui" -type f 2>/dev/null
```

## 📦 如果找不到文件

### 选项 A：重新从容器复制

```bash
# 1. 查看运行的容器
docker ps

# 输出示例：
# CONTAINER ID   IMAGE          COMMAND       CREATED        STATUS
# abc123def456   openhands...   "/bin/bash"   2 hours ago    Up 2 hours

# 2. 复制整个项目
docker cp abc123def456:/workspace/bproxy ~/bproxy

# 3. 进入目录
cd ~/bproxy
ls -la
```

### 选项 B：在容器内直接测试

```bash
# 1. 进入容器
docker exec -it <container-id> /bin/bash

# 2. 在容器内测试
cd /workspace/bproxy

# 3. 启动 Admin（后台）
./bin/admin -addr 0.0.0.0:8443 > admin.log 2>&1 &

# 4. 启动 Agent（后台）
./bin/agent -admin 127.0.0.1:8443 > agent.log 2>&1 &

# 5. 等待 2 秒
sleep 2

# 6. 查看日志
tail admin.log
tail agent.log

# 7. 测试（需要安装 curl）
curl -x socks5://127.0.0.1:1080 http://httpbin.org/ip
```

## 🚀 最简单的测试方法

### 在容器内一键测试

```bash
# 进入容器
docker exec -it <container-id> /bin/bash

# 运行测试脚本
cd /workspace/bproxy
./test-demo.sh
```

这个脚本会：
1. 自动启动 Admin
2. 自动启动 2 个 Agent
3. 显示日志
4. 按 Ctrl+C 停止

## 📝 检查 Docker 映射

查看您的 Docker 容器映射配置：

```bash
# 查看容器详情
docker inspect <container-id> | grep -A 10 "Mounts"

# 或者
docker inspect <container-id> | grep "Source\|Destination"
```

这会显示容器内路径和宿主机路径的映射关系。

## ❓ 常见问题

### Q1: 我的 Docker 映射目录是什么？

**A:** 执行以下命令查看：
```bash
docker inspect <container-id> | grep -A 5 "Mounts"
```

### Q2: 如何找到容器 ID？

**A:** 执行：
```bash
docker ps
```
第一列就是容器 ID。

### Q3: 文件权限问题

如果复制后文件没有执行权限：
```bash
chmod +x /path/to/bproxy/bin/*
```

### Q4: 找不到 Go 编译器

如果需要重新编译，但宿主机没有 Go：
- 选项 1：在容器内编译
- 选项 2：安装 Go：`sudo apt install golang-go`

## 📞 需要帮助？

请提供以下信息：

1. Docker 容器 ID：`docker ps`
2. 映射目录内容：`ls -la /opt/workspace_base/`
3. 是否能进入容器：`docker exec -it <id> /bin/bash`

---

**总结：**
- 容器内路径：`/workspace/bproxy`
- 宿主机路径：需要您确认映射目录
- 最简单：直接在容器内测试