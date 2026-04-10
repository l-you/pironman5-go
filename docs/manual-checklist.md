# Pironman 5 Manual Hardware Checklist

Run this on the real standard Pironman 5 after `go test ./...` passes in development.

1. Confirm device access:
   ```sh
   test -e /dev/i2c-1
   test -e /dev/spidev0.0
   test -e /sys/class/thermal/thermal_zone0/temp
   ```
2. Install and start the daemon:
   ```sh
   go build -o pironman5 ./cmd/pironman5
   sudo install -m 0755 pironman5 /usr/local/bin/pironman5
   sudo install -m 0644 systemd/pironman5.service /etc/systemd/system/pironman5.service
   sudo systemctl daemon-reload
   sudo systemctl restart pironman5.service
   systemctl status pironman5.service
   ```
3. Verify config and logging:
   ```sh
   sudo pironman5 -c
   sudo ls -l /etc/pironman5/config.json /var/log/pironman5/pironman5.log
   ```
4. Verify RGB:
   ```sh
   sudo pironman5 -re true
   sudo pironman5 -rs solid
   sudo pironman5 -rc ff6b00
   sudo systemctl restart pironman5.service
   ```
5. Verify OLED:
   ```sh
   sudo pironman5 -oe true
   sudo pironman5 -or 0
   sudo systemctl restart pironman5.service
   ```
   Confirm the display cycles through performance, IP, and disk pages.
6. Verify fan behavior:
   ```sh
   sudo pironman5 -gm 0
   sudo systemctl restart pironman5.service
   ```
   Confirm the GPIO fan follows upstream thresholds or Pi PWM state when PWM sysfs support is present.
