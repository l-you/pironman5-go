# pironman5-go

Go daemon essentials for the standard SunFounder Pironman 5.

This rewrite keeps the familiar `pironman5` command surface where it matters, but runs as a foreground Go daemon under systemd. V1 targets the standard Pironman 5 only: config, fan control, RGB, OLED pages, status sampling, logs, and service files. Dashboard/history, Mini/Max variants, vibration wake, and full installer automation are deferred.

## Build

```sh
go test ./...
go build -o pironman5 ./cmd/pironman5
```

## Install manually on Raspberry Pi OS

```sh
sudo install -m 0755 pironman5 /usr/local/bin/pironman5
sudo install -m 0644 systemd/pironman5.service /etc/systemd/system/pironman5.service
sudo mkdir -p /etc/pironman5 /var/log/pironman5
sudo systemctl daemon-reload
sudo systemctl enable --now pironman5.service
```

The default config path is `/etc/pironman5/config.json`. Use `pironman5 -cp /path/to/config.json ...` to override it.

## Compatible CLI examples

```sh
pironman5 -c
sudo pironman5 -rb 35
sudo pironman5 -rs solid
sudo pironman5 -rc ff6b00
sudo pironman5 -oe true
sudo pironman5 -or 180
sudo systemctl restart pironman5.service
```

`pironman5 start --background` is accepted for upstream compatibility, but the Go daemon stays in the foreground so systemd can supervise it with `Type=simple`.

## License

This project is intended to remain GPL-2.0-compatible because it uses the upstream GPL-2.0 Pironman projects as behavioral references.
