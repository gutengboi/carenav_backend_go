# Deploying to an Oracle Cloud VM

Assumes an Ubuntu VM (e.g. Oracle's Always Free Ampere instance) with Docker
installed and this repo checked out (or just the `Dockerfile` + `deploy/oracle/`
files copied over — the systemd unit doesn't need the Go source, only Docker).

## 1. Install Docker (one-time)

```bash
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker $USER   # log out/in after this
```

## 2. Build the image

From the repo root on the VM:

```bash
docker build -t carenav-backend-go:latest .
```

## 3. Set up the env file

```bash
sudo mkdir -p /etc/carenav
sudo cp deploy/oracle/api.env.example /etc/carenav/api.env
sudo chmod 600 /etc/carenav/api.env
sudo nano /etc/carenav/api.env   # fill in real secrets
```

## 4. Install the systemd service

```bash
sudo cp deploy/oracle/carenav-api.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now carenav-api
```

## 5. Verify

```bash
sudo systemctl status carenav-api
curl http://localhost:3000/
# → Welcome to my server
curl http://localhost:3000/swagger/index.html
```

Logs: `journalctl -u carenav-api -f`

## Opening the port

Oracle Cloud blocks inbound traffic by default at two layers — open port 3000
(or put Nginx/Caddy in front for 80/443 + TLS) in both:

- The VM's **Security List** / **Network Security Group** in the OCI console
- The instance's own firewall, e.g.:
  ```bash
  sudo iptables -I INPUT -p tcp --dport 3000 -j ACCEPT
  sudo netfilter-persistent save   # if installed
  ```

## Redeploying after a code change

```bash
git pull
docker build -t carenav-backend-go:latest .
sudo systemctl restart carenav-api
```
