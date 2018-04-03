// Build tags: only build this for the simulation build. Be sure to note the required blank line after.
// +build simulation

package obmc

import (
	"context"
	"fmt"
	"sync"

	"github.com/spf13/viper"
	"io/ioutil"
	// "github.com/go-yaml/yaml"
	yaml "gopkg.in/yaml.v2"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/utils"
	domain "github.com/superchalupa/go-redfish/src/redfishresource"

	plugins "github.com/superchalupa/go-redfish/src/ocp"
	"github.com/superchalupa/go-redfish/src/ocp/basicauth"
	"github.com/superchalupa/go-redfish/src/ocp/bmc"
	"github.com/superchalupa/go-redfish/src/ocp/chassis"
	"github.com/superchalupa/go-redfish/src/ocp/protocol"
	"github.com/superchalupa/go-redfish/src/ocp/root"
	"github.com/superchalupa/go-redfish/src/ocp/session"
	"github.com/superchalupa/go-redfish/src/ocp/system"
	"github.com/superchalupa/go-redfish/src/ocp/thermal"
	"github.com/superchalupa/go-redfish/src/ocp/thermal/fans"
	"github.com/superchalupa/go-redfish/src/ocp/thermal/temperatures"
)

func InitOCP(ctx context.Context, cfgMgr *viper.Viper, viperMu *sync.Mutex, ch eh.CommandHandler, eb eh.EventBus, ew *utils.EventWaiter) (*session.Service, *basicauth.Service, func()) {
	// initial implementation is one BMC, one Chassis, and one System.
	// Yes, this function is somewhat long, however there really isn't any logic here. If we start getting logic, this needs to be split.

	rootSvc, _ := root.New()

	sessionSvc, _ := session.New(
		session.Root(rootSvc),
	)
	basicAuthSvc, _ := basicauth.New()

	bmcSvc, _ := bmc.New(
		bmc.WithUniqueName("OBMC"),
	)

	prot, _ := protocol.New(
		protocol.WithBMC(bmcSvc),
	)

	chas, _ := chassis.New(
		chassis.AddManagedBy(bmcSvc),
		chassis.AddManagerInChassis(bmcSvc),
		chassis.WithUniqueName("1"),
	)

	bmcSvc.InChassis(chas)
	bmcSvc.AddManagerForChassis(chas)

	system, _ := system.New(
		system.WithUniqueName("1"),
		system.ManagedBy(bmcSvc),
		system.InChassis(chas),
	)

	bmcSvc.AddManagerForServer(system)
	chas.AddComputerSystem(system)

	therm, _ := thermal.New(
		thermal.InChassis(chas),
	)

	temps, _ := temperatures.New(
		temperatures.InThermal(therm),
		temperatures.WithSensor("inlet",
			&temperatures.RedfishThermalSensor{
				Name:                      "inlet temp sensor",
				SensorNumber:              0,
				ReadingCelsius:            22,
				UpperThresholdNonCritical: 100,
				UpperThresholdCritical:    150,
				UpperThresholdFatal:       200,
				MinReadingRangeTemp:       0,
				MaxReadingRangeTemp:       250,
				PhysicalContext:           "inlet",
			},
		),
		temperatures.WithSensor("ndc",
			&temperatures.RedfishThermalSensor{
				Name:                      "ndc temp sensor",
				SensorNumber:              1,
				ReadingCelsius:            23,
				UpperThresholdNonCritical: 100,
				UpperThresholdCritical:    150,
				UpperThresholdFatal:       200,
				MinReadingRangeTemp:       0,
				MaxReadingRangeTemp:       250,
				PhysicalContext:           "ndc",
			},
		),
	)

	fanObj, _ := fans.New(
		fans.InThermal(therm),
		fans.WithSensor("fan_X",
			&fans.RedfishFan{
				Name:            "inlet fan",
				PhysicalContext: "inlet",

				Reading:      2500,
				ReadingUnits: "RPM",

				UpperThresholdNonCritical: 3500,
				UpperThresholdCritical:    3600,
				UpperThresholdFatal:       3700,

				LowerThresholdNonCritical: 1000,
				LowerThresholdCritical:    900,
				LowerThresholdFatal:       800,

				MinReadingRange: 500,
				MaxReadingRange: 5000,
			},
		),
	)

	// VIPER Config:
	// pull the config from the YAML file to populate some static config options
	pullViperConfig := func() {
		fmt.Printf("\n\nDEBUG: proto: %s\n\n", cfgMgr.GetStringMapString("managers.OBMC.proto"))

		sessionSvc.ApplyOption(plugins.UpdateProperty("session_timeout", cfgMgr.GetInt("session.timeout")))

		for _, k := range []string{"name", "description", "model", "timezone", "version"} {
			bmcSvc.ApplyOption(plugins.UpdateProperty(k, cfgMgr.Get("managers.OBMC."+k)))
		}

		for k, _ := range cfgMgr.GetStringMapString("managers.OBMC.proto") {
			protocolOptions := map[string]interface{}{}

			options := cfgMgr.GetStringMap("managers.OBMC.proto." + k)
			fmt.Printf("DEBUG: options = %s\n\n", options)
			opts, ok := options["options"].(map[string]interface{})
			if ok {
				for _, vv := range opts {
					v, ok := vv.(map[string]interface{})
					if ok {
						protocolOptions[v["name"].(string)] = v["value"]
					}
				}
			}

			// TODO: better error checks on type assertions...
			prot.ApplyOption(protocol.WithProtocol(
				options["name"].(string),
				options["enabled"].(bool),
				options["port"].(int),
				protocolOptions))
		}

		for _, k := range []string{
			"name", "chassis_type", "model",
			"serial_number", "sku", "part_number",
			"asset_tag", "chassis_type", "manufacturer"} {
			chas.ApplyOption(plugins.UpdateProperty(k, cfgMgr.Get("chassis.1."+k)))
		}
		for _, k := range []string{
			"name", "system_type", "asset_tag", "manufacturer",
			"model", "serial_number", "sku", "The SKU", "part_number",
			"description", "power_state", "bios_version", "led", "system_hostname",
		} {
			system.ApplyOption(plugins.UpdateProperty(k, cfgMgr.Get("systems.1."+k)))
		}
	}
	pullViperConfig()

	cfgMgr.SetDefault("main.dumpConfigChanges.filename", "redfish-changed.yaml")
	cfgMgr.SetDefault("main.dumpConfigChanges.enabled", "true")
	dumpViperConfig := func() {
		viperMu.Lock()
		defer viperMu.Unlock()

		dumpFileName := cfgMgr.GetString("main.dumpConfigChanges.filename")
		enabled := cfgMgr.GetBool("main.dumpConfigChanges.enabled")
		if !enabled {
			return
		}

		// TODO: change this to a streaming write (reduce mem usage)
		var config map[string]interface{}
		cfgMgr.Unmarshal(&config)
		output, _ := yaml.Marshal(config)
		_ = ioutil.WriteFile(dumpFileName, output, 0644)
	}

	sessionSvc.AddPropertyObserver("session_timeout", func(newval interface{}) {
		viperMu.Lock()
		cfgMgr.Set("session.timeout", newval.(int))
		viperMu.Unlock()
		dumpViperConfig()
	})

	// register all of the plugins (do this first so we dont get any race
	// conditions if somebody accesses the URIs before these plugins are
	// registered
	domain.RegisterPlugin(func() domain.Plugin { return rootSvc })
	domain.RegisterPlugin(func() domain.Plugin { return sessionSvc })
	domain.RegisterPlugin(func() domain.Plugin { return basicAuthSvc })
	domain.RegisterPlugin(func() domain.Plugin { return bmcSvc })
	domain.RegisterPlugin(func() domain.Plugin { return prot })
	domain.RegisterPlugin(func() domain.Plugin { return chas })
	domain.RegisterPlugin(func() domain.Plugin { return system })
	domain.RegisterPlugin(func() domain.Plugin { return therm })
	domain.RegisterPlugin(func() domain.Plugin { return temps })
	domain.RegisterPlugin(func() domain.Plugin { return fanObj })

	// and now add everything to the URI tree
	rootSvc.AddResource(ctx, ch, eb, ew)
	sessionSvc.AddResource(ctx, ch, eb, ew)
	basicAuthSvc.AddResource(ctx, ch, eb, ew)
	bmcSvc.AddResource(ctx, ch, eb, ew)
	prot.AddResource(ctx, ch)
	chas.AddResource(ctx, ch)
	system.AddResource(ctx, ch, eb, ew)
	therm.AddResource(ctx, ch, eb, ew)
	temps.AddResource(ctx, ch, eb, ew)
	fanObj.AddResource(ctx, ch, eb, ew)

	bmcSvc.ApplyOption(plugins.UpdateProperty("manager.reset", func(event eh.Event, res *domain.HTTPCmdProcessedData) {
		fmt.Printf("Hello WORLD!\n\tGOT RESET EVENT\n")
		res.Results = map[string]interface{}{"RESET": "FAKE SIMULATED BMC RESET"}
	}))

	system.ApplyOption(plugins.UpdateProperty("computersystem.reset", func(event eh.Event, res *domain.HTTPCmdProcessedData) {
		fmt.Printf("Hello WORLD!\n\tGOT RESET EVENT\n")
		res.Results = map[string]interface{}{"RESET": "FAKE SIMULATED COMPUTER RESET"}
	}))

	return sessionSvc, basicAuthSvc, pullViperConfig
}
