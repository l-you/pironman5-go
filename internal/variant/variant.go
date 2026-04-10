package variant

import "github.com/l-you/pironman5-go/internal/config"

type Variant struct {
	Name           string
	ID             string
	ProductVersion string
	Peripherals    []string
	DTOverlays     []string
	DefaultConfig  config.System
}

func Standard() Variant {
	return Variant{
		Name:           "Pironman 5",
		ID:             "pironman5",
		ProductVersion: "",
		Peripherals: []string{
			"storage",
			"cpu",
			"network",
			"memory",
			"history",
			"log",
			"ws2812",
			"cpu_temperature",
			"gpu_temperature",
			"temperature_unit",
			"oled",
			"clear_history",
			"delete_log_file",
			"pwm_fan_speed",
			"gpio_fan_state",
			"gpio_fan_mode",
		},
		DTOverlays: []string{"sunfounder-pironman5.dtbo"},
		DefaultConfig: config.System{
			DataInterval:         1,
			DebugLevel:           "INFO",
			RGBColor:             "#0a1aff",
			RGBBrightness:        50,
			RGBStyle:             "breathing",
			RGBSpeed:             50,
			RGBEnable:            true,
			RGBLEDCount:          4,
			TemperatureUnit:      "C",
			OLEDEnable:           true,
			OLEDPageMode:         config.OLEDPageModeAuto,
			OLEDPage:             config.OLEDPagePerformance,
			OLEDRotation:         0,
			OLEDDisk:             "total",
			OLEDNetworkInterface: "all",
			GPIOFanPin:           6,
			GPIOFanMode:          0,
		},
	}
}
