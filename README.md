# Golang example SSH connection over Squid HTTP tunnel

Start squid

```bash
make squid
```

> Note: I needed to add `acl SSL_ports port 22` and `acl Safe_ports port 22` to squid configuration

Run app

```bash
go run main.go -addrhost <ssh-server-host-or-ip>
```