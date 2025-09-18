# MCP Gateway & Portal - Ubuntu Remote Deployment Guide
**Date**: September 18, 2025
**Target**: Ubuntu 20.04+ Remote Hosts
**Status**: Production Ready

## Quick Start

### 1. Clone and Configure
```bash
# On Ubuntu remote host
git clone https://github.com/jrmatherly/mcp-hub-gateway
cd mcp-hub-gateway

# Create environment configuration
cp .env.example .env
nano .env  # Edit with your production values
```

### 2. Deploy Services
```bash
# Make deployment script executable
chmod +x deploy-docker.sh

# Deploy all services
./deploy-docker.sh simple up

# Monitor deployment
./deploy-docker.sh simple logs
```

### 3. Verify Deployment
```bash
# Comprehensive health check
./deploy-docker.sh simple verify

# Check service status
./deploy-docker.sh simple status
```

### 4. Access Application
- **Frontend**: http://your-server-ip
- **Backend API**: http://your-server-ip:8080
- **Health Check**: http://your-server-ip/health

## Environment Configuration

### Required Variables (`.env` file)
```bash
# Authentication (Azure AD)
AZURE_TENANT_ID=your-tenant-id-here
AZURE_CLIENT_ID=your-client-id-here
AZURE_CLIENT_SECRET=your-client-secret-here

# Security (minimum 32 characters)
JWT_SECRET=your-very-long-jwt-secret-key-for-production

# Database
POSTGRES_DB=mcp_portal
POSTGRES_USER=postgres
POSTGRES_PASSWORD=secure-database-password
MCP_PORTAL_DATABASE_USERNAME=portal
MCP_PORTAL_DATABASE_PASSWORD=secure-portal-password

# Optional: Redis Authentication
REDIS_PASSWORD=redis-password-if-needed
```

## Ubuntu-Specific Features

### Docker Socket Handling
The containerization automatically handles Docker socket permissions on Ubuntu:

```bash
# Runtime detection and permission setup
DOCKER_GID=$(stat -c '%g' /var/run/docker.sock)
addgroup -g "$DOCKER_GID" docker || true
adduser portal docker || true
```

### System Service Integration
For persistent deployment, create a systemd service:

```bash
# Create service file
sudo tee /etc/systemd/system/mcp-portal.service <<EOF
[Unit]
Description=MCP Portal Docker Services
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/path/to/mcp-hub-gateway
ExecStart=/path/to/mcp-hub-gateway/deploy-docker.sh simple up
ExecStop=/path/to/mcp-hub-gateway/deploy-docker.sh simple down
User=your-user
Group=docker

[Install]
WantedBy=multi-user.target
EOF

# Enable and start service
sudo systemctl enable mcp-portal
sudo systemctl start mcp-portal
```

## Troubleshooting

### Common Ubuntu Issues

#### 1. Docker Socket Permission Denied
```bash
# Add user to docker group
sudo usermod -aG docker $USER
newgrp docker

# Verify access
docker info
```

#### 2. Port 80/443 Already in Use
```bash
# Check what's using the ports
sudo ss -tulpn | grep :80
sudo ss -tulpn | grep :443

# Stop conflicting services
sudo systemctl stop apache2  # or nginx
sudo systemctl stop nginx
```

#### 3. Memory Issues
```bash
# Check available memory
free -h

# Adjust Docker memory limits if needed
# Edit docker-compose.simple.yaml memory limits
```

#### 4. Build Issues on Ubuntu
```bash
# Clear all Docker cache
./deploy-docker.sh simple fix-cache

# Fix Node.js version issues
./deploy-docker.sh simple fix-node

# Debug deployment
./deploy-docker.sh simple debug
```

### Debug Commands

```bash
# Debug deployment issues
./deploy-docker.sh simple debug

# View service logs
./deploy-docker.sh simple logs

# Check service health
./deploy-docker.sh simple verify

# Monitor real-time logs
docker compose -f docker-compose.simple.yaml logs -f

# Check container resource usage
docker stats
```

## Security Configuration

### Firewall Setup (UFW)
```bash
# Enable firewall
sudo ufw enable

# Allow SSH (adjust port as needed)
sudo ufw allow 22/tcp

# Allow web traffic
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Optional: Allow backend API access
sudo ufw allow 8080/tcp
```

### SSL/TLS Configuration (Production)
For production deployment, configure SSL certificates:

```bash
# Install certbot
sudo apt update
sudo apt install snapd
sudo snap install --classic certbot

# Generate certificates
sudo certbot --nginx -d your-domain.com

# Update NGINX configuration in docker/nginx/conf.d/
```

## Monitoring and Maintenance

### Health Monitoring
```bash
# Continuous health monitoring script
cat <<'EOF' > /usr/local/bin/mcp-health-check.sh
#!/bin/bash
cd /path/to/mcp-hub-gateway
./deploy-docker.sh simple verify
if [ $? -ne 0 ]; then
    echo "Health check failed, restarting services..."
    ./deploy-docker.sh simple restart
fi
EOF

chmod +x /usr/local/bin/mcp-health-check.sh

# Add to crontab for regular checks
echo "*/5 * * * * /usr/local/bin/mcp-health-check.sh" | crontab -
```

### Log Management
```bash
# Configure log rotation
sudo tee /etc/logrotate.d/mcp-portal <<EOF
/var/lib/docker/volumes/mcp-portal-*-logs/_data/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 0644 root root
}
EOF
```

### Backup Procedures
```bash
# Database backup script
cat <<'EOF' > /usr/local/bin/mcp-backup.sh
#!/bin/bash
BACKUP_DIR="/backup/mcp-portal"
DATE=$(date +%Y%m%d_%H%M%S)
cd /path/to/mcp-hub-gateway

# Create backup directory
mkdir -p $BACKUP_DIR

# Backup database
docker compose -f docker-compose.simple.yaml exec postgres \
    pg_dump -U postgres mcp_portal > $BACKUP_DIR/database_$DATE.sql

# Backup volumes
docker run --rm -v mcp-portal-postgres-data:/data \
    -v $BACKUP_DIR:/backup alpine \
    tar czf /backup/postgres_data_$DATE.tar.gz -C /data .

# Remove old backups (keep 7 days)
find $BACKUP_DIR -name "*.sql" -mtime +7 -delete
find $BACKUP_DIR -name "*.tar.gz" -mtime +7 -delete
EOF

chmod +x /usr/local/bin/mcp-backup.sh

# Schedule daily backups
echo "0 2 * * * /usr/local/bin/mcp-backup.sh" | crontab -
```

## Performance Optimization

### Ubuntu-Specific Optimizations
```bash
# Increase file descriptor limits
echo "* soft nofile 65536" | sudo tee -a /etc/security/limits.conf
echo "* hard nofile 65536" | sudo tee -a /etc/security/limits.conf

# Optimize Docker daemon
sudo tee /etc/docker/daemon.json <<EOF
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  },
  "default-ulimits": {
    "nofile": {
      "Name": "nofile",
      "Hard": 65536,
      "Soft": 65536
    }
  }
}
EOF

sudo systemctl restart docker
```

### Resource Monitoring
```bash
# Install monitoring tools
sudo apt update
sudo apt install htop iotop nethogs

# Monitor containers
docker stats

# Monitor system resources
htop
```

## Updates and Maintenance

### Update Procedure
```bash
# 1. Backup current deployment
/usr/local/bin/mcp-backup.sh

# 2. Pull latest changes
git pull

# 3. Restart services
./deploy-docker.sh simple down
./deploy-docker.sh simple up

# 4. Verify deployment
./deploy-docker.sh simple verify
```

### Rollback Procedure
```bash
# 1. Stop current deployment
./deploy-docker.sh simple down

# 2. Checkout previous version
git checkout HEAD~1

# 3. Restore deployment
./deploy-docker.sh simple up

# 4. Verify rollback
./deploy-docker.sh simple verify
```

## Support and Troubleshooting

### Log Locations
- **Application Logs**: Docker container logs via `docker logs <container>`
- **System Logs**: `/var/log/syslog`
- **Docker Logs**: `journalctl -u docker.service`

### Common Commands
```bash
# Restart all services
./deploy-docker.sh simple restart

# View specific service logs
docker compose -f docker-compose.simple.yaml logs backend

# Connect to database
docker compose -f docker-compose.simple.yaml exec postgres \
    psql -U postgres -d mcp_portal

# Connect to Redis
docker compose -f docker-compose.simple.yaml exec redis redis-cli

# Clean up old containers and images
docker system prune -f

# Complete cleanup (removes data)
./deploy-docker.sh simple clean
```

## Production Checklist

### Pre-Deployment
- [ ] Ubuntu server with Docker installed
- [ ] Firewall configured
- [ ] DNS records configured
- [ ] SSL certificates obtained
- [ ] Environment variables configured
- [ ] Backup strategy implemented

### Post-Deployment
- [ ] Health checks passing
- [ ] SSL/TLS working
- [ ] Monitoring configured
- [ ] Backup tested
- [ ] Log rotation configured
- [ ] Performance baseline established

### Ongoing Maintenance
- [ ] Regular security updates
- [ ] Log monitoring
- [ ] Performance monitoring
- [ ] Backup verification
- [ ] Health check monitoring
- [ ] Certificate renewal

## Conclusion

The MCP Gateway & Portal is now fully containerized and ready for production deployment on Ubuntu hosts. The deployment process is automated, secure, and includes comprehensive monitoring and maintenance tools.

For additional support, refer to:
- **Main Documentation**: `/implementation-plan/README.md`
- **Containerization Analysis**: `/reports/CONTAINERIZATION_ANALYSIS_2025-09-18.md`
- **Security Guide**: `/docs/security.md`

The deployment eliminates complex installation scripts while providing enterprise-grade security, monitoring, and maintenance capabilities.