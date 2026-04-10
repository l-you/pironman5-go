package oled

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/l-you/pironman5-go/internal/config"
	"github.com/l-you/pironman5-go/internal/pbm"
	"github.com/l-you/pironman5-go/internal/status"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const (
	Width  = 128
	Height = 64
)

var autoPages = []string{
	config.OLEDPagePerformance,
	config.OLEDPageIP,
	config.OLEDPageDisk,
	config.OLEDPageHeart,
}

type Display interface {
	Display(context.Context, *image.Gray, int) error
	Clear(context.Context) error
	Off(context.Context) error
	Close() error
}

type Service struct {
	display Display
	sampler status.Sampler
	log     *slog.Logger
	mu      sync.RWMutex
	cfg     config.System
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	cache   imageCache
}

type imageCache struct {
	path    string
	modTime time.Time
	size    int64
	img     *image.Gray
	err     error
}

func New(display Display, sampler status.Sampler, cfg config.System, log *slog.Logger) *Service {
	return &Service{display: display, sampler: sampler, cfg: cfg, log: log.With("service", "oled")}
}

func (s *Service) Update(cfg config.System) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(s.cfg.OLEDImagePath) != strings.TrimSpace(cfg.OLEDImagePath) {
		s.cache = imageCache{}
	}
	s.cfg = cfg
}

func (s *Service) Start(parent context.Context) {
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(parent)
	s.cancel = cancel
	s.mu.Unlock()

	s.wg.Add(1)
	go s.loop(ctx)
}

func (s *Service) Stop() {
	s.mu.Lock()
	cancel := s.cancel
	s.cancel = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	s.wg.Wait()
	_ = s.display.Off(context.Background())
	_ = s.display.Close()
}

func (s *Service) loop(ctx context.Context) {
	defer s.wg.Done()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	pageIndex := 0
	lastPageChange := time.Now()
	for {
		cfg := s.snapshot()
		if !cfg.OLEDEnable {
			_ = s.display.Clear(ctx)
		} else {
			snap, err := s.sampler.Snapshot(ctx)
			if err != nil {
				s.log.Warn("sample oled status", "error", err)
			} else {
				pages := Pages(cfg)
				if pageIndex >= len(pages) {
					pageIndex = 0
				}
				if cfg.OLEDPageMode == config.OLEDPageModeAuto && time.Since(lastPageChange) >= 5*time.Second {
					pageIndex = (pageIndex + 1) % len(pages)
					lastPageChange = time.Now()
				}
				img := s.render(pages[pageIndex], snap, cfg)
				if err := s.display.Display(ctx, img, cfg.OLEDRotation); err != nil {
					s.log.Warn("display oled page", "error", err)
				}
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) snapshot() config.System {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func Pages(cfg config.System) []string {
	if cfg.OLEDPageMode == config.OLEDPageModeFixed {
		return []string{config.NormalizeOLEDPage(cfg.OLEDPage)}
	}
	pages := make([]string, len(autoPages), len(autoPages)+1)
	copy(pages, autoPages)
	if strings.TrimSpace(cfg.OLEDImagePath) != "" {
		pages = append(pages, config.OLEDPageImage)
	}
	return pages
}

func Render(page string, snap status.Snapshot, cfg config.System) *image.Gray {
	img := image.NewGray(image.Rect(0, 0, Width, Height))
	switch config.NormalizeOLEDPage(page) {
	case config.OLEDPageIP:
		renderIPs(img, snap, cfg)
	case config.OLEDPageDisk:
		renderDisks(img, snap, cfg)
	case config.OLEDPageHeart:
		renderHeart(img)
	case config.OLEDPageImage:
		path := strings.TrimSpace(cfg.OLEDImagePath)
		loaded, err := loadImage(path)
		renderConfiguredImage(img, path, loaded, err)
	default:
		renderPerformance(img, snap, cfg)
	}
	return img
}

func (s *Service) render(page string, snap status.Snapshot, cfg config.System) *image.Gray {
	if config.NormalizeOLEDPage(page) != config.OLEDPageImage {
		return Render(page, snap, cfg)
	}
	img := image.NewGray(image.Rect(0, 0, Width, Height))
	path := strings.TrimSpace(cfg.OLEDImagePath)
	loaded, err := s.loadCachedImage(path)
	renderConfiguredImage(img, path, loaded, err)
	return img
}

func renderPerformance(img *image.Gray, snap status.Snapshot, cfg config.System) {
	cpu := clampPercent(snap.CPUPercent)
	memory := clampPercent(snap.MemoryPercent)
	drawText(img, "CPU", 0, 0)
	drawText(img, fmt.Sprintf("%3.0f%%", cpu), 0, 14)
	drawBar(img, 0, 28, 56, 7, cpu)
	drawTextClipped(img, temperatureLabel(snap.CPUTempC, cfg.TemperatureUnit), 72, 0, 8)
	drawText(img, "FAN", 72, 18)
	drawText(img, fmt.Sprintf("%d", snap.FanRPM), 72, 32)
	drawText(img, "RAM", 0, 42)
	drawTextClipped(img, fmt.Sprintf("%s/%s", status.FormatBytesString(snap.MemoryUsed), status.FormatBytesString(snap.MemoryTotal)), 32, 42, 13)
	drawBar(img, 0, 56, 126, 7, memory)
}

func renderIPs(img *image.Gray, snap status.Snapshot, cfg config.System) {
	drawText(img, "IP", 0, 0)
	ips := selectedIPs(snap, cfg)
	if len(ips) == 0 {
		drawText(img, "NO IPv4", 0, 24)
		return
	}
	for i, ip := range ips {
		if i >= 2 {
			break
		}
		y := 14 + i*25
		drawTextClipped(img, ip.Interface, 0, y, 18)
		drawTextClipped(img, ip.Address, 0, y+12, 18)
	}
}

func renderDisks(img *image.Gray, snap status.Snapshot, cfg config.System) {
	drawText(img, "DISK", 0, 0)
	disks := selectedDisks(snap, cfg)
	if len(disks) == 0 {
		drawText(img, "NO MOUNTS", 0, 24)
		return
	}
	if len(disks) == 1 {
		disk := disks[0]
		drawTextClipped(img, fmt.Sprintf("%s %3.0f%%", disk.Name, clampPercent(disk.Percent)), 0, 16, 18)
		drawTextClipped(img, formatDiskUsage(disk), 0, 32, 18)
		drawBar(img, 0, 50, 126, 7, clampPercent(disk.Percent))
		return
	}
	for i, disk := range disks {
		if i >= 2 {
			break
		}
		y := 14 + i*25
		drawTextClipped(img, fmt.Sprintf("%s %3.0f%%", disk.Name, clampPercent(disk.Percent)), 0, y, 18)
		drawTextClipped(img, formatDiskUsage(disk), 0, y+12, 18)
	}
}

func renderHeart(img *image.Gray) {
	heart := []string{
		"011001100",
		"111111111",
		"111111111",
		"111111111",
		"011111110",
		"001111100",
		"000111000",
		"000010000",
	}
	scale := 5
	x0 := (Width - len(heart[0])*scale) / 2
	y0 := 5
	for row, line := range heart {
		for col, pixel := range line {
			if pixel != '1' {
				continue
			}
			fillRect(img, x0+col*scale, y0+row*scale, scale, scale)
		}
	}
	drawText(img, "PIRONMAN", 36, 49)
}

func renderConfiguredImage(dst *image.Gray, path string, src *image.Gray, err error) {
	if path == "" {
		drawText(dst, "NO IMAGE", 0, 24)
		drawText(dst, "SET -oj", 0, 38)
		return
	}
	if err != nil {
		drawText(dst, "IMAGE ERR", 0, 20)
		drawTextClipped(dst, err.Error(), 0, 36, 18)
		return
	}
	drawImage(dst, src)
}

func (s *Service) loadCachedImage(path string) (*image.Gray, error) {
	if path == "" {
		return nil, nil
	}
	info, err := os.Stat(path)
	if err != nil {
		s.cache = imageCache{path: path, err: fmt.Errorf("stat: %w", err)}
		return nil, s.cache.err
	}
	if s.cache.path == path && s.cache.size == info.Size() && s.cache.modTime.Equal(info.ModTime()) {
		return s.cache.img, s.cache.err
	}
	img, err := loadImage(path)
	s.cache = imageCache{
		path:    path,
		modTime: info.ModTime(),
		size:    info.Size(),
		img:     img,
		err:     err,
	}
	return img, err
}

func loadImage(path string) (*image.Gray, error) {
	if path == "" {
		return nil, nil
	}
	img, err := pbm.DecodeFile(path)
	if err != nil {
		return nil, err
	}
	if img.Bounds().Dx() != Width || img.Bounds().Dy() != Height {
		return nil, fmt.Errorf("pbm must be %dx%d", Width, Height)
	}
	return img, nil
}

func drawImage(dst, src *image.Gray) {
	if src == nil {
		return
	}
	draw.Draw(dst, dst.Bounds(), src, src.Bounds().Min, draw.Src)
}

func selectedIPs(snap status.Snapshot, cfg config.System) []status.IP {
	iface := strings.TrimSpace(cfg.OLEDNetworkInterface)
	if iface == "" || strings.EqualFold(iface, "all") {
		return snap.IPs
	}
	out := make([]status.IP, 0, 1)
	for _, ip := range snap.IPs {
		if ip.Interface == iface {
			out = append(out, ip)
		}
	}
	return out
}

func selectedDisks(snap status.Snapshot, cfg config.System) []status.Disk {
	target := strings.TrimSpace(cfg.OLEDDisk)
	if target == "" || strings.EqualFold(target, "total") {
		return aggregateDisks(snap.Disks)
	}
	out := make([]status.Disk, 0, 1)
	for _, disk := range snap.Disks {
		if disk.Name == target || disk.Mount == target {
			out = append(out, disk)
		}
	}
	return out
}

func aggregateDisks(disks []status.Disk) []status.Disk {
	var total uint64
	var used uint64
	for _, disk := range disks {
		total += disk.Total
		used += disk.Used
	}
	if total == 0 {
		return nil
	}
	return []status.Disk{{Name: "total", Total: total, Used: used, Percent: float64(used) * 100 / float64(total)}}
}

func temperatureLabel(tempC float64, unit string) string {
	labelUnit := strings.ToUpper(unit)
	temp := tempC
	if labelUnit == "F" {
		temp = tempC*9/5 + 32
	} else {
		labelUnit = "C"
	}
	return fmt.Sprintf("%3.0f deg%s", temp, labelUnit)
}

func formatDiskUsage(disk status.Disk) string {
	_, unit := status.FormatBytes(disk.Total)
	used, _ := status.FormatBytes(disk.Used, unit)
	total, _ := status.FormatBytes(disk.Total, unit)
	return fmt.Sprintf("%s/%s%s", compactFloat(used), compactFloat(total), unit)
}

func compactFloat(value float64) string {
	if value >= 10 || value == float64(int64(value)) {
		return fmt.Sprintf("%.0f", value)
	}
	return fmt.Sprintf("%.1f", value)
}

func drawText(img *image.Gray, text string, x, y int) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.White),
		Face: basicfont.Face7x13,
		Dot:  fixed.P(x, y+12),
	}
	d.DrawString(text)
}

func drawTextClipped(img *image.Gray, text string, x, y, maxChars int) {
	drawText(img, truncateText(text, maxChars), x, y)
}

func truncateText(text string, maxChars int) string {
	if maxChars <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= maxChars {
		return text
	}
	if maxChars <= 2 {
		return string(runes[:maxChars])
	}
	return string(runes[:maxChars-2]) + ".."
}

func drawBar(img *image.Gray, x, y, width, height int, percent float64) {
	drawRect(img, x, y, width, height, false)
	fill := int(float64(width-2) * clampPercent(percent) / 100)
	if fill > 0 {
		fillRect(img, x+1, y+1, fill, height-2)
	}
}

func drawRect(img *image.Gray, x, y, width, height int, fill bool) {
	for yy := y; yy < y+height; yy++ {
		for xx := x; xx < x+width; xx++ {
			if fill || yy == y || yy == y+height-1 || xx == x || xx == x+width-1 {
				setWhite(img, xx, yy)
			}
		}
	}
}

func fillRect(img *image.Gray, x, y, width, height int) {
	drawRect(img, x, y, width, height, true)
}

func setWhite(img *image.Gray, x, y int) {
	if image.Pt(x, y).In(img.Bounds()) {
		img.SetGray(x, y, color.Gray{Y: 255})
	}
}

func clampPercent(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}
