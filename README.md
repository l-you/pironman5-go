# pironman5-go
Go daemon essentials for the standard SunFounder Pironman 5.

This rewrite keeps the familiar `pironman5` command surface where it matters, but runs as a foreground Go daemon under systemd. V1 targets the standard Pironman 5 only: config, fan control, RGB, OLED pages, status sampling, logs, and service files. Dashboard/history, Mini/Max variants, vibration wake, and full installer automation are deferred.

> [!IMPORTANT]
> This is the only text in this repository written without AI.
> 
> This project is a Go rewrite of the essential service for the SunFounder Pironman 5. The original source code is written in Python: https://github.com/sunfounder/pironman5
> 
> I created and use this project for personal purposes.
> 
> You may use https://github.com/l-you/pironman5-go for any purpose, but I do not provide any guarantees regarding its functionality.
> 
> I do not promise proper versioning, CI, or verified binary builds.
> I can only say that:
> - I use it myself;
> - I use the binary stored in the `dist` directory;
> - the binary is compiled with the `go build` command.
> 
> If you are a careful developer, you should not download or run this binary directly.
> It is safer to treat the commit hash as the version and compile the program from source.
> That is a much safer practice. Do not trust binaries from the internet.
> 
> I personally use it on my Raspberry Pi 5 + Pironman 5 server. I have not tested it on the Pironman 5 Max, so compatibility is unknown.
> 
> Why did I make this project? I did not like the Python RAM overhead on my lightweight server. It also consumed a small amount of CPU.
> 
> Numbers:
> - The original Python-based Pironman 5 service used approximately 70–110 MB of RAM.
> - The same service, rewritten in Go, uses a stable ~10 MB of RAM.
> 
> The initial version of this repository does not include the dashboard feature. I may add support for it later. I expect it would require about 5 MB of additional RAM, which is not a big deal.
> 
> CPU usage dropped from 0.02% to 0.00% for general server stats while running Debian OS + the Pironman service.


## Build

```sh
go test ./...
go build -o pironman5 ./cmd/pironman5
go build -o pironman5-image ./cmd/pironman5-image
```

## Replace an existing Pironman service from git clone

Run this on Debian or Raspberry Pi OS arm64 on the Raspberry Pi 5. Do not keep the old Python Pironman service running at the same time as this daemon, because both services can try to control the same fan, RGB, and OLED devices.

```sh
sudo apt update
sudo apt install -y git

git clone https://github.com/l-you/pironman5-go.git
cd pironman5-go

systemctl list-unit-files '*pironman*'
sudo systemctl disable --now pironman5.service || true
sudo cp /etc/pironman5/config.json /etc/pironman5/config.json.bak 2>/dev/null || true

sudo install -m 0755 dist/pironman5-linux-arm64 /usr/local/bin/pironman5
sudo install -m 0755 dist/pironman5-image-linux-arm64 /usr/local/bin/pironman5-image
sudo install -m 0644 systemd/pironman5.service /etc/systemd/system/pironman5.service
sudo mkdir -p /etc/pironman5 /var/log/pironman5
sudo systemctl daemon-reload
sudo systemctl enable --now pironman5.service
sudo systemctl status pironman5.service
```

If the old unit has a different name, replace `pironman5.service` with the name shown by `systemctl list-unit-files '*pironman*'`.

To rebuild from source on the Pi instead of using the committed binary, install Go 1.26.1 or newer and run:

```sh
go test ./...
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags '-s -w' -o dist/pironman5-linux-arm64 ./cmd/pironman5
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags '-s -w' -o dist/pironman5-image-linux-arm64 ./cmd/pironman5-image
```

## Install manually on Raspberry Pi OS

```sh
sudo install -m 0755 pironman5 /usr/local/bin/pironman5
sudo install -m 0755 pironman5-image /usr/local/bin/pironman5-image
sudo install -m 0644 systemd/pironman5.service /etc/systemd/system/pironman5.service
sudo mkdir -p /etc/pironman5 /var/log/pironman5
sudo systemctl daemon-reload
sudo systemctl enable --now pironman5.service
```

The default config path is `/etc/pironman5/config.json`. Use `pironman5 -cp /path/to/config.json ...` to override it.

## Common commands

### Show config

Prints the active config file.

```sh
pironman5 -c
```

### Use another config file

Reads or writes a custom config path instead of `/etc/pironman5/config.json`.

```sh
pironman5 -cp /path/to/config.json -c
```

### Set RGB color to yellow

Sets RGB to yellow, then restarts the daemon so the running service uses the new color.

```sh
sudo pironman5 -rc ffff00
sudo systemctl restart pironman5.service
```

### Set RGB brightness

Sets brightness to 35 percent.

```sh
sudo pironman5 -rb 35
sudo systemctl restart pironman5.service
```

### Set RGB style

Sets the RGB effect to a steady color.

```sh
sudo pironman5 -rs solid
sudo systemctl restart pironman5.service
```

Supported styles: `solid`, `breathing`, `flow`, `flow_reverse`, `rainbow`, `rainbow_reverse`, `hue_cycle`.

### Set RGB animation speed

Changes animation duration. Fading and hue effects still render at a fixed frame cadence.

```sh
sudo pironman5 -rp 75
sudo systemctl restart pironman5.service
```

### Change GPIO fan mode

Shows the current mode, then turns the GPIO fan on from LOW and above.

```sh
pironman5 -gm
sudo pironman5 -gm 1
sudo systemctl restart pironman5.service
```

Modes:

```text
0 = GPIO fan always on
1 = on from LOW and above
2 = on from MEDIUM and above
3 = on only at HIGH
4 = always off
```

### Change GPIO fan pin

Sets the GPIO fan control pin to BCM 6.

```sh
sudo pironman5 -gp 6
sudo systemctl restart pironman5.service
```

### Enable OLED

Turns OLED pages on.

```sh
sudo pironman5 -oe true
sudo systemctl restart pironman5.service
```

### Auto-switch OLED screens

Cycles through `performance`, `ip`, `disk`, and `heart` screens. Adds `image` when `-oj` points to one or more `.pbm` files.

```sh
sudo pironman5 -om auto
sudo systemctl restart pironman5.service
```

### Show one OLED screen

Shows only the heart screen.

```sh
sudo pironman5 -om fixed -op heart
sudo systemctl restart pironman5.service
```

### Convert a JPG or PNG for the OLED

Creates a 128x64 monochrome PBM for the daemon.

```sh
pironman5-image convert /home/pi/oled.jpg /home/pi/oled.pbm
```

### Show one custom OLED image

The daemon reads PBM only.

```sh
sudo pironman5 -om fixed -op image -oj /home/pi/oled.pbm
sudo systemctl restart pironman5.service
```

### Rotate multiple OLED images

Switches between multiple PBM images every 3 seconds.

```sh
sudo pironman5 -om fixed -op image -oj /home/pi/1.pbm,/home/pi/2.pbm,/home/pi/3.pbm -ot 3
sudo systemctl restart pironman5.service
```

Screens: `performance`, `ip`, `disk`, `heart`, `image`.

### Rotate OLED

Rotates the OLED output by 180 degrees.

```sh
sudo pironman5 -or 180
sudo systemctl restart pironman5.service
```

`pironman5 start --background` is accepted for upstream compatibility, but the Go daemon stays in the foreground so systemd can supervise it with `Type=simple`.

## License

This project is intended to remain GPL-2.0-compatible because it uses the upstream GPL-2.0 Pironman projects as behavioral references.
