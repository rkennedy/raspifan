# Raspifan

I wrote this to control the fan on my Raspberry Pi 4. There are other similar
tools, but some didn't seem to work, or they required installing runtimes for
languages I didn't want to maintain on my server.

Writing my own provided me a chance to learn about writing a Systemd service in
Go.

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
