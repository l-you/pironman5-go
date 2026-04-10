package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/l-you/pironman5-go/internal/buildinfo"
	"github.com/l-you/pironman5-go/internal/config"
	"github.com/l-you/pironman5-go/internal/daemon"
	"github.com/l-you/pironman5-go/internal/variant"
)

type OptionalString struct {
	Set      bool
	HasValue bool
	Value    string
}

type Options struct {
	Command              string
	Version              bool
	ShowConfig           bool
	Background           bool
	ConfigPath           OptionalString
	DebugLevel           OptionalString
	RGBColor             OptionalString
	RGBBrightness        OptionalString
	RGBStyle             OptionalString
	RGBSpeed             OptionalString
	RGBEnable            OptionalString
	RGBLEDCount          OptionalString
	TemperatureUnit      OptionalString
	GPIOFanMode          OptionalString
	GPIOFanPin           OptionalString
	OLEDEnable           OptionalString
	OLEDPageMode         OptionalString
	OLEDPage             OptionalString
	OLEDImagePath        OptionalString
	OLEDImageInterval    OptionalString
	OLEDDisk             OptionalString
	OLEDNetworkInterface OptionalString
	OLEDRotation         OptionalString
}

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printHelp(stdout)
		return nil
	}
	opts, err := Parse(args)
	if err != nil {
		return err
	}
	if opts.Version {
		fmt.Fprintln(stdout, buildinfo.Version)
		return nil
	}

	configPath := buildinfo.DefaultConfigPath
	if opts.ConfigPath.Set {
		if !opts.ConfigPath.HasValue {
			fmt.Fprintf(stdout, "Config path: %s\n", configPath)
			return nil
		}
		configPath = opts.ConfigPath.Value
	}

	v := variant.Standard()
	cfg, err := config.LoadOrCreate(configPath, v.DefaultConfig)
	if err != nil {
		return err
	}

	changed, printed, err := applyConfigOptions(stdout, &cfg, opts)
	if err != nil {
		return err
	}
	if changed {
		if err := config.Save(configPath, cfg); err != nil {
			return err
		}
	}
	if opts.ShowConfig {
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, string(data))
		return nil
	}
	if opts.Command == "" {
		if !changed && !printed && !opts.Background {
			printHelp(stdout)
		}
		return nil
	}

	switch opts.Command {
	case "start":
		runtime, err := daemon.New(configPath)
		if err != nil {
			return err
		}
		return runtime.Run(ctx)
	case "stop":
		return daemon.StopPID(buildinfo.PIDPath)
	case "restart":
		if err := daemon.StopPID(buildinfo.PIDPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(stderr, "warning: %v\n", err)
		}
		time.Sleep(2 * time.Second)
		runtime, err := daemon.New(configPath)
		if err != nil {
			return err
		}
		return runtime.Run(ctx)
	default:
		return fmt.Errorf("unknown command %q", opts.Command)
	}
}

func Parse(args []string) (Options, error) {
	var opts Options
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if isCommand(arg) {
			if opts.Command != "" {
				return Options{}, fmt.Errorf("multiple commands: %s and %s", opts.Command, arg)
			}
			opts.Command = arg
			continue
		}
		name, inline, hasInline := splitInline(arg)
		switch name {
		case "-v", "--version":
			opts.Version = true
		case "-c", "--config":
			opts.ShowConfig = true
		case "--background":
			opts.Background = true
		case "-cp", "--config-path":
			opts.ConfigPath = takeOptional(args, &i, inline, hasInline)
		case "-dl", "--debug-level":
			opts.DebugLevel = takeOptional(args, &i, inline, hasInline)
		case "-rc", "--rgb-color":
			opts.RGBColor = takeOptional(args, &i, inline, hasInline)
		case "-rb", "--rgb-brightness":
			opts.RGBBrightness = takeOptional(args, &i, inline, hasInline)
		case "-rs", "--rgb-style":
			opts.RGBStyle = takeOptional(args, &i, inline, hasInline)
		case "-rp", "--rgb-speed":
			opts.RGBSpeed = takeOptional(args, &i, inline, hasInline)
		case "-re", "--rgb-enable":
			opts.RGBEnable = takeOptional(args, &i, inline, hasInline)
		case "-rl", "--rgb-led-count":
			opts.RGBLEDCount = takeOptional(args, &i, inline, hasInline)
		case "-u", "--temperature-unit":
			opts.TemperatureUnit = takeOptional(args, &i, inline, hasInline)
		case "-gm", "--gpio-fan-mode":
			opts.GPIOFanMode = takeOptional(args, &i, inline, hasInline)
		case "-gp", "--gpio-fan-pin":
			opts.GPIOFanPin = takeOptional(args, &i, inline, hasInline)
		case "-oe", "--oled-enable":
			opts.OLEDEnable = takeOptional(args, &i, inline, hasInline)
		case "-om", "--oled-page-mode":
			opts.OLEDPageMode = takeOptional(args, &i, inline, hasInline)
		case "-op", "--oled-page":
			opts.OLEDPage = takeOptional(args, &i, inline, hasInline)
		case "-oj", "--oled-image-path":
			opts.OLEDImagePath = takeOptional(args, &i, inline, hasInline)
		case "-ot", "--oled-image-interval":
			opts.OLEDImageInterval = takeOptional(args, &i, inline, hasInline)
		case "-od", "--oled-disk":
			opts.OLEDDisk = takeOptional(args, &i, inline, hasInline)
		case "-oi", "--oled-network-interface":
			opts.OLEDNetworkInterface = takeOptional(args, &i, inline, hasInline)
		case "-or", "--oled-rotation":
			opts.OLEDRotation = takeOptional(args, &i, inline, hasInline)
		case "-rd", "--remove-dashboard":
			return Options{}, fmt.Errorf("dashboard removal is deferred in the Go v1 daemon")
		default:
			return Options{}, fmt.Errorf("unknown argument %q", arg)
		}
	}
	return opts, nil
}

func applyConfigOptions(out io.Writer, cfg *config.File, opts Options) (bool, bool, error) {
	changed := false
	printed := false
	applyString := func(flag OptionalString, label string, current string, set func(string)) error {
		if !flag.Set {
			return nil
		}
		if !flag.HasValue {
			fmt.Fprintf(out, "%s: %s\n", label, current)
			printed = true
			return nil
		}
		set(flag.Value)
		fmt.Fprintf(out, "Set %s: %s\n", label, flag.Value)
		changed = true
		return nil
	}
	applyInt := func(flag OptionalString, label string, current int, min int, max int, set func(int)) error {
		if !flag.Set {
			return nil
		}
		if !flag.HasValue {
			fmt.Fprintf(out, "%s: %d\n", label, current)
			printed = true
			return nil
		}
		value, err := strconv.Atoi(flag.Value)
		if err != nil || value < min || value > max {
			return fmt.Errorf("invalid %s: expected integer between %d and %d", label, min, max)
		}
		set(value)
		fmt.Fprintf(out, "Set %s: %d\n", label, value)
		changed = true
		return nil
	}
	applyBool := func(flag OptionalString, label string, current bool, set func(bool)) error {
		if !flag.Set {
			return nil
		}
		if !flag.HasValue {
			fmt.Fprintf(out, "%s: %t\n", label, current)
			printed = true
			return nil
		}
		value, ok := parseBool(flag.Value)
		if !ok {
			return fmt.Errorf("invalid %s: expected true/false/on/off/1/0", label)
		}
		set(value)
		fmt.Fprintf(out, "Set %s: %t\n", label, value)
		changed = true
		return nil
	}
	applyImagePaths := func(flag OptionalString, current []string) error {
		if !flag.Set {
			return nil
		}
		if !flag.HasValue {
			fmt.Fprintf(out, "OLED image paths: %s\n", strings.Join(current, ", "))
			printed = true
			return nil
		}
		paths := config.ParseOLEDImagePaths(flag.Value)
		if len(paths) == 0 {
			return fmt.Errorf("invalid OLED image paths: expected one or more comma-separated .pbm paths")
		}
		cfg.System.OLEDImagePaths = paths
		cfg.System.OLEDImagePath = paths[0]
		fmt.Fprintf(out, "Set OLED image paths: %s\n", strings.Join(paths, ", "))
		changed = true
		return nil
	}

	if err := applyString(opts.DebugLevel, "Debug level", cfg.System.DebugLevel, func(v string) { cfg.System.DebugLevel = strings.ToUpper(v) }); err != nil {
		return false, printed, err
	}
	if opts.RGBColor.Set && opts.RGBColor.HasValue {
		if _, err := config.ParseHexColor(opts.RGBColor.Value); err != nil {
			return false, printed, err
		}
	}
	if err := applyString(opts.RGBColor, "RGB color", cfg.System.RGBColor, func(v string) { cfg.System.RGBColor = v }); err != nil {
		return false, printed, err
	}
	if err := applyInt(opts.RGBBrightness, "RGB brightness", cfg.System.RGBBrightness, 0, 100, func(v int) { cfg.System.RGBBrightness = v }); err != nil {
		return false, printed, err
	}
	if opts.RGBStyle.Set && opts.RGBStyle.HasValue && !slices.Contains(config.RGBStyles, opts.RGBStyle.Value) {
		return false, printed, fmt.Errorf("invalid RGB style: expected one of %v", config.RGBStyles)
	}
	if err := applyString(opts.RGBStyle, "RGB style", cfg.System.RGBStyle, func(v string) { cfg.System.RGBStyle = v }); err != nil {
		return false, printed, err
	}
	if err := applyInt(opts.RGBSpeed, "RGB speed", cfg.System.RGBSpeed, 0, 100, func(v int) { cfg.System.RGBSpeed = v }); err != nil {
		return false, printed, err
	}
	if err := applyBool(opts.RGBEnable, "RGB enable", cfg.System.RGBEnable, func(v bool) { cfg.System.RGBEnable = v }); err != nil {
		return false, printed, err
	}
	if err := applyInt(opts.RGBLEDCount, "RGB LED count", cfg.System.RGBLEDCount, 1, 256, func(v int) { cfg.System.RGBLEDCount = v }); err != nil {
		return false, printed, err
	}
	if opts.TemperatureUnit.Set && opts.TemperatureUnit.HasValue {
		unit := strings.ToUpper(opts.TemperatureUnit.Value)
		if unit != "C" && unit != "F" {
			return false, printed, fmt.Errorf("invalid temperature unit: expected C or F")
		}
		opts.TemperatureUnit.Value = unit
	}
	if err := applyString(opts.TemperatureUnit, "Temperature unit", cfg.System.TemperatureUnit, func(v string) { cfg.System.TemperatureUnit = strings.ToUpper(v) }); err != nil {
		return false, printed, err
	}
	if err := applyInt(opts.GPIOFanMode, "GPIO fan mode", cfg.System.GPIOFanMode, 0, 4, func(v int) { cfg.System.GPIOFanMode = v }); err != nil {
		return false, printed, err
	}
	if err := applyInt(opts.GPIOFanPin, "GPIO fan pin", cfg.System.GPIOFanPin, 0, 40, func(v int) { cfg.System.GPIOFanPin = v }); err != nil {
		return false, printed, err
	}
	if err := applyBool(opts.OLEDEnable, "OLED enable", cfg.System.OLEDEnable, func(v bool) { cfg.System.OLEDEnable = v }); err != nil {
		return false, printed, err
	}
	if opts.OLEDPageMode.Set && opts.OLEDPageMode.HasValue {
		mode := strings.ToLower(opts.OLEDPageMode.Value)
		if !slices.Contains(config.OLEDPageModes, mode) {
			return false, printed, fmt.Errorf("invalid OLED page mode: expected one of %v", config.OLEDPageModes)
		}
		opts.OLEDPageMode.Value = mode
	}
	if err := applyString(opts.OLEDPageMode, "OLED page mode", cfg.System.OLEDPageMode, func(v string) { cfg.System.OLEDPageMode = strings.ToLower(v) }); err != nil {
		return false, printed, err
	}
	if opts.OLEDPage.Set && opts.OLEDPage.HasValue {
		page := config.NormalizeOLEDPage(opts.OLEDPage.Value)
		if !slices.Contains(config.OLEDPages, page) {
			return false, printed, fmt.Errorf("invalid OLED page: expected one of %v", config.OLEDPages)
		}
		opts.OLEDPage.Value = page
	}
	if err := applyString(opts.OLEDPage, "OLED page", cfg.System.OLEDPage, func(v string) { cfg.System.OLEDPage = config.NormalizeOLEDPage(v) }); err != nil {
		return false, printed, err
	}
	if err := applyImagePaths(opts.OLEDImagePath, cfg.System.OLEDImages()); err != nil {
		return false, printed, err
	}
	if err := applyInt(opts.OLEDImageInterval, "OLED image interval", cfg.System.OLEDImageInterval, 1, 86400, func(v int) { cfg.System.OLEDImageInterval = v }); err != nil {
		return false, printed, err
	}
	if err := applyString(opts.OLEDDisk, "OLED disk", cfg.System.OLEDDisk, func(v string) { cfg.System.OLEDDisk = v }); err != nil {
		return false, printed, err
	}
	if err := applyString(opts.OLEDNetworkInterface, "OLED network interface", cfg.System.OLEDNetworkInterface, func(v string) { cfg.System.OLEDNetworkInterface = v }); err != nil {
		return false, printed, err
	}
	if err := applyInt(opts.OLEDRotation, "OLED rotation", cfg.System.OLEDRotation, 0, 180, func(v int) { cfg.System.OLEDRotation = v }); err != nil {
		return false, printed, err
	}
	if err := cfg.System.Validate(); err != nil {
		return false, printed, err
	}
	return changed, printed, nil
}

func isCommand(arg string) bool {
	return arg == "start" || arg == "restart" || arg == "stop"
}

func splitInline(arg string) (string, string, bool) {
	if strings.HasPrefix(arg, "--") {
		name, value, ok := strings.Cut(arg, "=")
		return name, value, ok
	}
	return arg, "", false
}

func takeOptional(args []string, index *int, inline string, hasInline bool) OptionalString {
	if hasInline {
		return OptionalString{Set: true, HasValue: true, Value: inline}
	}
	if *index+1 < len(args) && !strings.HasPrefix(args[*index+1], "-") {
		*index++
		return OptionalString{Set: true, HasValue: true, Value: args[*index]}
	}
	return OptionalString{Set: true}
}

func parseBool(value string) (bool, bool) {
	switch strings.ToLower(value) {
	case "true", "1", "on", "yes", "y":
		return true, true
	case "false", "0", "off", "no", "n":
		return false, true
	default:
		return false, false
	}
}

func printHelp(out io.Writer) {
	fmt.Fprintln(out, `Usage: pironman5 [command] [flags]

Commands:
  start      Run the daemon in the foreground
  stop       Stop the daemon using /run/pironman5.pid
  restart    Stop and then run the daemon in the foreground

Core flags:
  -v, --version
  -c, --config
  -cp, --config-path [path]
  -dl, --debug-level [debug|info|warning|error|critical]
  --background    Accepted for upstream compatibility; no-op in Go v1

RGB flags:
  -rc, --rgb-color [RRGGBB]
  -rb, --rgb-brightness [0-100]
  -rs, --rgb-style [solid|breathing|flow|flow_reverse|rainbow|rainbow_reverse|hue_cycle]
  -rp, --rgb-speed [0-100]
  -re, --rgb-enable [true|false]
  -rl, --rgb-led-count [count]

Fan/OLED flags:
  -u,  --temperature-unit [C|F]
  -gm, --gpio-fan-mode [0-4]
  -gp, --gpio-fan-pin [pin]
  -oe, --oled-enable [true|false]
  -om, --oled-page-mode [auto|fixed]
  -op, --oled-page [performance|ip|disk|heart|image]
  -oj, --oled-image-path [path1.pbm,path2.pbm]
  -ot, --oled-image-interval [seconds]
  -od, --oled-disk [total|device]
  -oi, --oled-network-interface [all|interface]
  -or, --oled-rotation [0|180]`)
}
