package fans

import (
	"context"

	"github.com/superchalupa/go-redfish/src/log"
	"github.com/superchalupa/go-redfish/src/ocp/view"
	domain "github.com/superchalupa/go-redfish/src/redfishresource"

	eh "github.com/looplab/eventhorizon"
)

// So... this class is set up in a somewhat interesting way to support having
// Fan.Slot.N objects both as PowerSupplies/PSU.Slot.N as well as in the main
// Power object.

func AddAggregate(ctx context.Context, logger log.Logger, v *view.View, ch eh.CommandHandler) map[string]interface{} {
	ch.HandleCommand(
		ctx,
		&domain.CreateRedfishResource{
			ID:          v.GetUUID(),
			Collection:  false,
			ResourceURI: v.GetURI(),
			Type:        "#Thermal.v1_0_2.Fan",
			Context:     "/redfish/v1/$metadata#Thermal.Thermal",
			Privileges: map[string]interface{}{
				"GET":    []string{"Login"},
				"POST":   []string{}, // cannot create sub objects
				"PUT":    []string{},
				"PATCH":  []string{"ConfigureManager"},
				"DELETE": []string{}, // can't be deleted
			},
			Properties: GetViewFragment(v),
		})

	return GetViewFragment(v)
}

func GetViewFragment(v *view.View) map[string]interface{} {
	return map[string]interface{}{
		"@odata.type":    "#Thermal.v1_0_2.Fan",
		"@odata.context": "/redfish/v1/$metadata#Thermal.Thermal",
		"@odata.id":      v.GetURI(),
		"Description":    "Represents the properties for Fan and Cooling",
		"FanName@meta":   v.Meta(view.PropGET("name")),
		"MemberId@meta":  v.Meta(view.PropGET("unique_id")),

		// TODO: Henry - fan_controller now populates sensor readings, but don't have units in that data. where are they/
		"ReadingUnits@meta": v.Meta(view.PropGET("reading_units")),
		// TODO: Henry - putting the fan pwm in here for now to prove it's
		// working. Fixup to match odatalite. Also need to fix pump because
		// it's not formatting this properly
		"Reading@meta": v.Meta(view.PropGET("fanpwm")),

		// TODO: Henry - fan_controller has a health status as part of the fan object. probably need to interpret that and drop into model
		"Status": map[string]interface{}{
			"HealthRollup": "TEST_VALUE",
			"Health@meta":  v.Meta(view.PropGET("health")),
		},
		"Oem": map[string]interface{}{
			"ReadingUnits@meta":    v.Meta(view.PropGET("oem_reading_units")),
			"Reading@meta":         v.Meta(view.PropGET("oem_reading")),
			"FirmwareVersion@meta": v.Meta(view.PropGET("firmware_version")),
			"HardwareVersion@meta": v.Meta(view.PropGET("hardware_version")),
			"GraphicsURI@meta":     v.Meta(view.PropGET("graphics_uri")),
			"Attributes@meta":      v.Meta(view.GETProperty("attributes"), view.GETFormatter("attributeFormatter"), view.GETModel("default"), view.PropPATCH("attributes", "ar_dump")),
		},
	}
}
