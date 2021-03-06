package registry

import (
	"context"

	"github.com/superchalupa/go-redfish/src/log"
	"github.com/superchalupa/go-redfish/src/ocp/view"
	domain "github.com/superchalupa/go-redfish/src/redfishresource"

	eh "github.com/looplab/eventhorizon"
)

func AddAggregate(ctx context.Context, logger log.Logger, v *view.View, ch eh.CommandHandler, eb eh.EventBus) {
	ch.HandleCommand(
		ctx,
		&domain.CreateRedfishResource{
			ID:          v.GetUUID(),
			Collection:  false,
			ResourceURI: v.GetURI(),
			Type:        "#MessageRegistryFile.v1_0_2.MessageRegistryFile",
			Context:     "/redfish/v1/$metadata#MessageRegistryFile.MessageRegistryFile",
			Privileges: map[string]interface{}{
				"GET":    []string{"Login"},
				"POST":   []string{}, // cannot create sub objects
				"PUT":    []string{},
				"PATCH":  []string{},
				"DELETE": []string{}, // can't be deleted
			},
			Properties: map[string]interface{}{
				"Description":   "Base Message Registry File locations",
				"Id@meta":       v.Meta(view.PropGET("registry_id")),
				"Name@meta":     v.Meta(view.PropGET("registry_name")),
				"Registry@meta": v.Meta(view.PropGET("registry_type")),
			}})
}
