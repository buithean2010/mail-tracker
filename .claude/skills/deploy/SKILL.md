---
description: Deploy to production Azure VM with Docker Compose and SSL
when_to_use: When the user wants to deploy, redeploy, or roll back production, or asks about Azure VM setup, SSL certificates, or production environment
allowed-tools: Bash(docker compose -f docker-compose.prod.yml *), Bash(git pull *), Bash(git checkout *), Bash(curl -f *)
argument-hint: "[--build] [service-name]"
---

# Deploy to Production

## Prerequisites on Azure VM
```bash
# Docker + Docker Compose
sudo apt update && sudo apt install docker.io docker-compose-v2 -y
sudo usermod -aG docker $USER

# Firewall
sudo ufw allow 80 && sudo ufw allow 443 && sudo ufw enable
```

## SSL Certificate (Let's Encrypt)
```bash
sudo apt install certbot -y
sudo certbot certonly --standalone -d domain.com
# Cert files: /etc/letsencrypt/live/domain.com/
# Auto-renew: certbot renew (add to cron or systemd timer)
```

## First deploy
```bash
# Clone repo
git clone <repo> && cd mail-tracker/src

# Create production .env
cp .env.example .env
# Fill in: DB_PASSWORD, AES_KEY (32 bytes hex), CLIENT_ID, CLIENT_SECRET, SESSION_SECRET

# Build + start
docker compose -f docker-compose.prod.yml up -d --build
```

## Update / Redeploy
```bash
git pull
docker compose -f docker-compose.prod.yml up -d --build backend frontend
# Postgres + nginx rarely need a rebuild
```

## Verify after deploy
```bash
docker compose -f docker-compose.prod.yml ps
docker compose -f docker-compose.prod.yml logs backend --tail=50
curl -f https://domain.com/api/healthz
```

## Rollback
```bash
git checkout <previous-tag>
docker compose -f docker-compose.prod.yml up -d --build
```

**Do NOT use** `docker compose down -v` in production — it will delete all PostgreSQL data.
