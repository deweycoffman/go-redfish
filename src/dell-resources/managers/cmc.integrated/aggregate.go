package cmc_integrated

// this file should define the BMC Manager object golang data structures where
// we put all the data, plus the aggregate that pulls the data.  actual data
// population should happen in an impl class. ie. no dbus calls in this file

import (
	"context"

	"github.com/superchalupa/go-redfish/src/eventwaiter"
	"github.com/superchalupa/go-redfish/src/log"
	"github.com/superchalupa/go-redfish/src/ocp/health"
	"github.com/superchalupa/go-redfish/src/ocp/view"
	domain "github.com/superchalupa/go-redfish/src/redfishresource"

	eh "github.com/looplab/eventhorizon"
)

type waiter interface {
	Listen(context.Context, func(eh.Event) bool) (*eventwaiter.EventListener, error)
}

func AddAggregate(ctx context.Context, logger log.Logger, v *view.View, ch eh.CommandHandler) *view.View {

	properties := map[string]interface{}{
		"Id@meta": v.Meta(view.PropGET("unique_name")),
		//"Name@meta": v.Meta(view.PropGET("name")),
		"Name": "Manager", //hardcoded in odatalite
		// TODO: is this in AR somewhere?
		"ManagerType": "BMC", //hardcoded in odatalite
		//"Description@meta":         v.Meta(view.PropGET("description")),
		"Description":              "BMC", //hardcoded in odatalite
		"Model@meta":               v.Meta(view.PropGET("model")),
		"DateTime@meta":            map[string]interface{}{"GET": map[string]interface{}{"plugin": "datetime"}}, //taken from local system clock (WORK ON THIS)
		"DateTimeLocalOffset@meta": v.Meta(view.PropGET("timezone")),                                            //taken from local system clock
		"FirmwareVersion@meta":     v.Meta(view.PropGET("firmware_version")),
		"Links": map[string]interface{}{
			//"ManagerForServers@meta": v.Meta(view.PropGET("bmc_manager_for_servers")), // not in odatalite json?
			// "ManagerForChassis@odata.count": 1,
			"ManagerForChassis@meta":             v.Meta(view.PropGET("manager_for_chassis")),
			"ManagerForChassis@odata.count@meta": v.Meta(view.PropGET("manager_for_chassis_count")),
			//"ManagerInChassis@meta":  v.Meta(view.PropGET("in_chassis")), //not in odatalite json?
		},

		"SerialConsole": map[string]interface{}{
			"ConnectTypesSupported@odata.count@meta": v.Meta(view.PropGET("connect_types_supported_count")),
			"MaxConcurrentSessions":                  0,
			"ConnectTypesSupported@meta":             v.Meta(view.PropGET("connect_types_supported")),
			"ServiceEnabled":                         false,
		},

		"CommandShell": map[string]interface{}{
			"ConnectTypesSupported@odata.count@meta": v.Meta(view.PropGET("connect_types_supported_count")),
			"MaxConcurrentSessions":                  0,
			"ConnectTypesSupported@meta":             v.Meta(view.PropGET("connect_types_supported")),
			"ServiceEnabled":                         false,
		},

		"LogServices": map[string]interface{}{
			"@odata.id": v.GetURI() + "/LogServices",
		},

		"GraphicalConsole": map[string]interface{}{
			"ConnectTypesSupported@odata.count@meta": v.Meta(view.PropGET("connect_types_supported_count")),
			"MaxConcurrentSessions":                  0,
			"ConnectTypesSupported@meta":             v.Meta(view.PropGET("connect_types_supported")),
			"ServiceEnabled":                         false,
		},

		"Oem": map[string]interface{}{
			"@odata.type": "#DellManager.v1_0_0.DellManager",
			"OemAttributes": map[string]interface{}{
				"@odata.id": v.GetURI() + "/Attributes",
			},
			"CertificateService": map[string]interface{}{
				"@odata.id": v.GetURI() + "/CertificateService",
			},
		},

		"Actions": map[string]interface{}{
			"#Manager.Reset": map[string]interface{}{
				"target": v.GetActionURI("manager.reset"),
				"ResetType@Redfish.AllowableValues": []string{
					"GracefulRestart",
				},
			},
			"Oem": map[string]interface{}{
				// TODO: Remove per JIT-66996
				"DellManager.v1_0_0#DellManager.ResetToDefaults": map[string]interface{}{
					"ResetType@Redfish.AllowableValues": []string{
						"ClearToShip",
						"Decommission",
						"ResetFactoryConfig",
						"ResetToEngineeringDefaults",
					},
					"target": v.GetActionURI("manager.resettodefaults"),
				},
				"#Manager.ForceFailover": map[string]interface{}{
					"target": v.GetActionURI("manager.forcefailover"),
				},
				"#DellManager.v1_0_0.DellManager.ResetToDefaults": map[string]interface{}{
					"ResetType@Redfish.AllowableValues": []string{
						"ClearToShip",
						"Decommission",
						"ResetFactoryConfig",
						"ResetToEngineeringDefaults",
					},
					"target": v.GetActionURI("manager.resettodefaults"),
				},
				"OemManager.v1_0_0#OemManager.ExportSystemConfiguration": map[string]interface{}{
					"ExportFormat@Redfish.AllowableValues": []string{
						"XML",
						"JSON",
					},
					"ExportUse@Redfish.AllowableValues": []string{
						"Default",
						"Clone",
						"Replace",
					},
					"OemManager.v1_0_0#OemManager.ExportSystemConfiguration": []string{
						"Default",
						"IncludeReadOnly",
						"IncludePasswordHashValues",
						"IncludeReadOnly,IncludePasswordHashValues",
					},
					"ShareParameters": map[string]interface{}{
						"IgnoreCertificateWarning@Redfish.AllowableValues": []string{
							"Disabled",
							"Enabled",
						},
						"ProxySupport@Redfish.AllowableValues": []string{
							"Disabled",
							"EnabledProxyDefault",
							"Enabled",
						},
						"ProxyType@Redfish.AllowableValues": []string{
							"HTTP",
							"SOCKS4",
						},
						"ShareParameters@Redfish.AllowableValues": []string{
							"IPAddress",
							"ShareName",
							"FileName",
							"UserName",
							"Password",
							"Workgroup",
							"ProxyServer",
							"ProxyUserName",
							"ProxyPassword",
							"ProxyPort",
						},
						"ShareType@Redfish.AllowableValues": []string{
							"NFS",
							"CIFS",
							"HTTP",
							"HTTPS",
						},
						"Target@Redfish.AllowableValues": []string{
							"ALL",
							"IDRAC",
							"BIOS",
							"NIC",
							"RAID",
						},
					},
					"target": v.GetActionURI("manager.exportsystemconfig"),
				},
				"OemManager.v1_0_0#OemManager.ImportSystemConfiguration": map[string]interface{}{
					"HostPowerState@Redfish.AllowableValues": []string{
						"On",
						"Off",
					},
					"ImportSystemConfiguration@Redfish.AllowableValues": []string{
						"TimeToWait",
						"ImportBuffer",
					},
					"ShareParameters": map[string]interface{}{
						"IgnoreCertificateWarning@Redfish.AllowableValues": []string{
							"Disabled",
							"Enabled",
						},
						"ProxySupport@Redfish.AllowableValues": []string{
							"Disabled",
							"EnabledProxyDefault",
							"Enabled",
						},
						"ProxyType@Redfish.AllowableValues": []string{
							"HTTP",
							"SOCKS4",
						},
						"ShareParameters@Redfish.AllowableValues": []string{
							"IPAddress",
							"ShareName",
							"FileName",
							"UserName",
							"Password",
							"Workgroup",
							"ProxyServer",
							"ProxyUserName",
							"ProxyPassword",
							"ProxyPort",
						},
						"ShareType@Redfish.AllowableValues": []string{
							"NFS",
							"CIFS",
							"HTTP",
							"HTTPS",
						},
						"Target@Redfish.AllowableValues": []string{
							"ALL",
							"IDRAC",
							"BIOS",
							"NIC",
							"RAID",
						},
					},
					"ShutdownType@Redfish.AllowableValues": []string{
						"Graceful",
						"Forced",
						"NoReboot",
					},
					"target": v.GetActionURI("manager.importsystemconfig"),
				},
				"OemManager.v1_0_0#OemManager.ImportSystemConfigurationPreview": map[string]interface{}{
					"ImportSystemConfigurationPreview@Redfish.AllowableVaues": []string{
						"ImportBuffer",
					},
					"ShareParameters": map[string]interface{}{
						"IgnoreCertificateWarning@Redfish.AllowableValues": []string{
							"Disabled",
							"Enabled",
						},
						"ProxySupport@Redfish.AllowableValues": []string{
							"Disabled",
							"EnabledProxyDefault",
							"Enabled",
						},
						"ProxyType@Redfish.AllowableValues": []string{
							"HTTP",
							"SOCKS4",
						},
						"ShareParameters@Redfish.AllowableValues": []string{
							"IPAddress",
							"ShareName",
							"FileName",
							"UserName",
							"Password",
							"Workgroup",
							"ProxyServer",
							"ProxyUserName",
							"ProxyPassword",
							"ProxyPort",
						},
						"ShareType@Redfish.AllowableValues": []string{
							"NFS",
							"CIFS",
							"HTTP",
							"HTTPS",
						},
						"Target@Redfish.AllowableValues": []string{
							"ALL",
						},
					},
					"target": v.GetActionURI("manager.importsystemconfigpreview"),
				},
			},
		},
	}

	// TODO: move this out
	redundancy := map[string]interface{}{
		"@odata.type": "#Redundancy.v1_0_2.Redundancy",
		"RedundancySet": []interface{}{
			map[string]interface{}{
				"@odata.id": "/redfish/v1/Managers/CMC.Integrated.1",
			},
			map[string]interface{}{
				"@odata.id": "/redfish/v1/Managers/CMC.Integrated.2",
			},
		},
		"Name": "ManagerRedundancy",
		"RedundancySet@odata.count": 2, //is supposed to be length of redundancy set
		"@odata.id":                 "/redfish/v1/Managers/CMC.Integrated.1#Redundancy",
		"@odata.context":            "/redfish/v1/$metadata#Redundancy.Redundancy",
		"Mode@meta":                 v.Meta(view.PropGET("redundancy_mode")),
		"MinNumNeeded@meta":         v.Meta(view.PropGET("redundancy_min")),
		"MaxNumSupported@meta":      v.Meta(view.PropGET("redundancy_max")),
	}
	health.GetHealthFragment(v, "health", properties)
	health.GetHealthFragment(v, "redundancy_health", redundancy)
	properties["Redundancy"] = []interface{}{redundancy}
	properties["Redundancy@odata.count"] = len(properties["Redundancy"].([]interface{}))

	ch.HandleCommand(
		ctx,
		&domain.CreateRedfishResource{
			ID:          v.GetUUID(),
			Collection:  false,
			ResourceURI: v.GetURI(),
			Type:        "#Manager.v1_0_2.Manager",
			Context:     "/redfish/v1/$metadata#Manager.Manager",
			Privileges: map[string]interface{}{
				"GET":    []string{"Login"},
				"POST":   []string{}, // cannot create sub objects
				"PUT":    []string{},
				"PATCH":  []string{"ConfigureManager"},
				"DELETE": []string{}, // can't be deleted
			},
			Properties: properties,
		})

	return v
}
