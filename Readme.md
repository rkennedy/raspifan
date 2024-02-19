# Install
```bash
sudo ./install.bash
```

This copies the binary to /usr/local/lib and creates a corresponding .service
file for systemd to consume. It does not activate the service, so afterward,
run this:

```bash
sudo systemctl enable --now raspifan.service
```
