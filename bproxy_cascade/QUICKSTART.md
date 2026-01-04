# BProxy Quick Start Guide

## 🚀 5-Minute Setup

### Prerequisites
- Linux system (Ubuntu/Debian/Kali recommended)
- Go 1.21+ installed
- Root access (for L3 proxy features)

### Installation

```bash
# Clone or navigate to BProxy directory
cd /workspace/bproxy

# Build all binaries
make build

# Verify build
ls -lh bin/
# Should show: admin, admin-tui, agent
```

## 🎯 Basic Usage

### Scenario 1: Single Agent Connection

**Terminal 1 - Start Admin Server:**
```bash
cd /workspace/bproxy
./bin/admin -addr 0.0.0.0:8443
```

**Terminal 2 - Start Agent:**
```bash
cd /workspace/bproxy
./bin/agent -admin 127.0.0.1:8443
```

**Expected Output (Admin):**
```
2026/01/03 17:00:00 Admin server listening on 0.0.0.0:8443
2026/01/03 17:00:05 Agent registered: abc123... (hostname: myhost, IPs: [192.168.1.10])
2026/01/03 17:00:20 Heartbeat from abc123...
```

**Expected Output (Agent):**
```
2026/01/03 17:00:05 Starting BProxy Agent...
2026/01/03 17:00:05 Connecting to admin at 127.0.0.1:8443
2026/01/03 17:00:05 Agent abc123... connected to admin
```

### Scenario 2: TUI Mode (Recommended)

**Terminal 1 - Start Admin with TUI:**
```bash
cd /workspace/bproxy
./bin/admin-tui -addr 0.0.0.0:8443
```

**Terminal 2 - Start Agent:**
```bash
cd /workspace/bproxy
./bin/agent -admin 127.0.0.1:8443
```

**TUI Interface:**
```
┌─────────────────────────────────────────────────────────────┐
│  🔥 BProxy - Red Team Proxy Tool 🔥                        │
├──────────────────────────┬──────────────────────────────────┤
│  📡 Agent Topology       │  💻 Console                      │
│                          │                                  │
│  ▶ ● abc12345 myhost     │  Active Connections: 1           │
│      ↳ 192.168.1.10      │                                  │
│      ↳ Last seen: 2s ago │  Recent Activity:                │
│                          │    BProxy Admin Console - Ready  │
│                          │    Agent registered              │
│                          │                                  │
└──────────────────────────┴──────────────────────────────────┘
│  Press 'h' for help | 'q' to quit                          │
└─────────────────────────────────────────────────────────────┘
```

**TUI Controls:**
- `↑/k` - Move up
- `↓/j` - Move down
- `Enter` - Select node
- `r` - Refresh
- `h` - Help
- `q` - Quit

## 🔧 Configuration Options

### Admin Server

```bash
# Default (all interfaces, port 8443)
./bin/admin -addr 0.0.0.0:8443

# Specific interface
./bin/admin -addr 192.168.1.100:8443

# Custom TLS certificates
./bin/admin -addr 0.0.0.0:8443 -cert server.crt -key server.key

# TUI mode
./bin/admin-tui -addr 0.0.0.0:8443
```

### Agent Client

```bash
# Connect to local admin
./bin/agent -admin 127.0.0.1:8443

# Connect to remote admin
./bin/agent -admin 10.0.0.5:8443

# Run in background
nohup ./bin/agent -admin 10.0.0.5:8443 > agent.log 2>&1 &
```

## 🧪 Testing

### Quick Test Script

```bash
cd /workspace/bproxy
./test-demo.sh
```

This will:
1. Build all binaries
2. Start admin server
3. Start 2 agents
4. Show logs
5. Keep running until Ctrl+C

### Manual Testing

**Test 1: Connection**
```bash
# Terminal 1
./bin/admin -addr 0.0.0.0:8443

# Terminal 2
./bin/agent -admin 127.0.0.1:8443

# Verify: Admin shows "Agent registered"
```

**Test 2: Multiple Agents**
```bash
# Terminal 1
./bin/admin-tui -addr 0.0.0.0:8443

# Terminal 2
./bin/agent -admin 127.0.0.1:8443

# Terminal 3
./bin/agent -admin 127.0.0.1:8443

# Verify: TUI shows 2 agents
```

**Test 3: Heartbeat**
```bash
# Start admin and agent
# Wait 15 seconds
# Verify: Admin logs show "Heartbeat from..."
```

**Test 4: Reconnection**
```bash
# Start admin and agent
# Kill agent (Ctrl+C)
# Wait 60 seconds
# Verify: Admin marks agent as dead
# Restart agent
# Verify: Agent reconnects
```

## 📊 Monitoring

### Check Logs

```bash
# Admin logs
tail -f admin.log

# Agent logs
tail -f agent.log

# Filter for errors
grep ERROR admin.log
```

### Check Connections

```bash
# Active connections to admin
netstat -an | grep 8443

# Process status
ps aux | grep bproxy
```

### Check TLS

```bash
# Test TLS connection
openssl s_client -connect 127.0.0.1:8443

# Verify certificate
openssl s_client -connect 127.0.0.1:8443 -showcerts
```

## 🐛 Troubleshooting

### Problem: Agent won't connect

**Check 1: Network connectivity**
```bash
telnet <admin-ip> 8443
# or
nc -zv <admin-ip> 8443
```

**Check 2: Firewall**
```bash
# Check firewall rules
sudo iptables -L -n | grep 8443

# Allow port (if needed)
sudo iptables -A INPUT -p tcp --dport 8443 -j ACCEPT
```

**Check 3: Admin is running**
```bash
sudo netstat -tlnp | grep 8443
```

### Problem: TUI not displaying correctly

**Solution 1: Set terminal type**
```bash
export TERM=xterm-256color
./bin/admin-tui -addr 0.0.0.0:8443
```

**Solution 2: Check terminal size**
```bash
# Terminal should be at least 80x24
stty size
```

**Solution 3: Use screen/tmux**
```bash
tmux
./bin/admin-tui -addr 0.0.0.0:8443
```

### Problem: Permission denied

**For TUN interface (L3 proxy):**
```bash
# Run with sudo
sudo ./bin/admin-tui -addr 0.0.0.0:8443

# Or set capabilities
sudo setcap cap_net_admin=eip ./bin/admin-tui
```

### Problem: Build fails

**Solution:**
```bash
# Clean and rebuild
make clean
make deps
make build

# Check Go version
go version  # Should be 1.21+

# Update dependencies
go mod tidy
```

## 🎓 Next Steps

### Learn More
1. Read `README.md` for full documentation
2. Check `EXAMPLES.md` for 10+ usage scenarios
3. Study `ARCHITECTURE.md` for technical details
4. Review `PROJECT_SUMMARY.md` for overview

### Advanced Features
1. Multi-level cascading
2. Layer 3 routing proxy
3. Custom TLS certificates
4. Topology management
5. Message relay

### Development
1. Add new message types
2. Extend TUI interface
3. Implement SOCKS5 proxy
4. Add port forwarding
5. Create plugins

## 📝 Common Commands

```bash
# Build
make build

# Clean
make clean

# Run admin (CLI)
make run-admin

# Run admin (TUI)
make run-admin-tui

# Run agent
make run-agent

# Install system-wide
sudo make install

# Then use:
bproxy-admin -addr 0.0.0.0:8443
bproxy-agent -admin <ip>:8443
```

## 🔒 Security Notes

1. **Always use TLS**: Never disable encryption
2. **Secure admin port**: Use firewall rules
3. **Rotate certificates**: Generate new certs regularly
4. **Monitor connections**: Watch for unauthorized agents
5. **Use strong passwords**: If adding authentication
6. **Audit logs**: Review activity regularly
7. **Limit exposure**: Don't expose admin to internet
8. **Update regularly**: Keep BProxy updated

## 💡 Tips & Tricks

### Tip 1: Run in Background
```bash
# Admin
nohup ./bin/admin -addr 0.0.0.0:8443 > admin.log 2>&1 &

# Agent
nohup ./bin/agent -admin <ip>:8443 > agent.log 2>&1 &
```

### Tip 2: Systemd Service
```bash
# Create /etc/systemd/system/bproxy-admin.service
[Unit]
Description=BProxy Admin Server
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/bproxy-admin -addr 0.0.0.0:8443
Restart=always

[Install]
WantedBy=multi-user.target

# Enable and start
sudo systemctl enable bproxy-admin
sudo systemctl start bproxy-admin
```

### Tip 3: Docker Deployment
```bash
# Build image
docker build -t bproxy-admin -f Dockerfile.admin .

# Run container
docker run -d -p 8443:8443 bproxy-admin
```

### Tip 4: SSH Tunneling
```bash
# Forward admin port through SSH
ssh -L 8443:localhost:8443 user@admin-server

# Connect agent to forwarded port
./bin/agent -admin 127.0.0.1:8443
```

### Tip 5: Multiple Admins
```bash
# Run on different ports
./bin/admin -addr 0.0.0.0:8443  # Admin 1
./bin/admin -addr 0.0.0.0:8444  # Admin 2
```

## 🎯 Success Checklist

- [ ] Built all binaries successfully
- [ ] Admin server starts without errors
- [ ] Agent connects to admin
- [ ] TUI displays agent information
- [ ] Heartbeat messages appear in logs
- [ ] Agent reconnects after disconnect
- [ ] Multiple agents can connect
- [ ] TUI updates in real-time

## 📞 Getting Help

1. Check logs for error messages
2. Review troubleshooting section
3. Read full documentation
4. Check GitHub issues (if applicable)
5. Review architecture documentation

## 🎉 You're Ready!

You now have a working BProxy installation. Start experimenting with:
- Multiple agents
- Network topologies
- TUI interface
- Advanced features

Happy hacking! 🔥