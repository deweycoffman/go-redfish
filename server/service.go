package redfishserver

import (
	"context"
	"errors"
	eh "github.com/superchalupa/eventhorizon"
	commandbus "github.com/superchalupa/eventhorizon/commandbus/local"
	repo "github.com/superchalupa/eventhorizon/repo/memory"
	"strings"

	"github.com/superchalupa/go-rfs/domain"

	"fmt"
)

// Service is the business logic for a redfish server
type Service interface {
	RawJSONRedfishGet(ctx context.Context, pathTemplate, url string, args map[string]string) (interface{}, error)
}

// ServiceMiddleware is a chainable behavior modifier for Service.
type ServiceMiddleware func(Service) Service

var (
	// ErrNotFound is returned when a request isnt present (404)
	ErrNotFound = errors.New("not found")
)

// Config is where we store the current service data
type config struct {
	baseURI   string
	verURI    string
	treeID    eh.UUID
	cmdbus    *commandbus.CommandBus
	odataRepo *repo.Repo
}

func (c *config) makeFullyQualifiedV1(path string) string {
	return c.baseURI + "/" + c.verURI + "/" + path
}

// NewService is how we initialize the business logic
func NewService(baseURI string, commandbus *commandbus.CommandBus, repo *repo.Repo, id eh.UUID) Service {
	cfg := config{baseURI: baseURI, verURI: "v1", cmdbus: commandbus, odataRepo: repo, treeID: id}
	go cfg.startup()
	return &cfg
}

func (rh *config) RawJSONRedfishGet(ctx context.Context, pathTemplate, url string, args map[string]string) (output interface{}, err error) {
	noHashPath := strings.SplitN(url, "#", 2)[0]

	rawTree, err := rh.odataRepo.Find(ctx, rh.treeID)
	if err != nil {
		fmt.Printf("could not find tree: %s\n", err.Error())
		return nil, ErrNotFound
	}

	tree, ok := rawTree.(*domain.OdataTree)
	if !ok {
		fmt.Printf("somehow it wasnt a tree! %s\n", err.Error())
		return nil, ErrNotFound
	}

	requested, err := rh.odataRepo.Find(ctx, tree.Tree[noHashPath])
	if err != nil {
		return nil, ErrNotFound
	}
	item, ok := requested.(*domain.OdataResource)
	return item.Properties, nil
}

func (rh *config) startup() {
	ctx := context.Background()
	rh.createTreeLeaf(ctx, "/redfish/", map[string]interface{}{"v1": "/redfish/v1/"})
	rh.createTreeLeaf(ctx, "/redfish/v1/",
		map[string]interface{}{
			"@odata.context": "/redfish/v1/$metadata#ServiceRoot",
			"@odata.id":      "/redfish/v1",
			"@odata.type":    "#ServiceRoot.v1_0_2.ServiceRoot",
			"Description":    "Root Service",
			"Id":             "RootService",
			"Name":           "Root Service",
			"RedfishVersion": "v1_0_2",
			"Systems":        map[string]interface{}{"@odata.id": "/redfish/v1/Systems"},
			"Chassis":        map[string]interface{}{"@odata.id": "/redfish/v1/Chassis"},
			"EventService":   map[string]interface{}{"@odata.id": "/redfish/v1/EventService"},
			"JsonSchemas":    map[string]interface{}{"@odata.id": "/redfish/v1/JSONSchemas"},
			"Managers":       map[string]interface{}{"@odata.id": "/redfish/v1/Managers"},
			"Registries":     map[string]interface{}{"@odata.id": "/redfish/v1/Registries"},
			"SessionService": map[string]interface{}{"@odata.id": "/redfish/v1/SessionService"},
			"Tasks":          map[string]interface{}{"@odata.id": "/redfish/v1/TaskService"},
			"UpdateService":  map[string]interface{}{"@odata.id": "/redfish/v1/UpdateService"},
			"Links": map[string]interface{}{
				"Sessions": map[string]interface{}{"@odata.id": "/redfish/v1/Sessions"},
			},
			"AccountService": map[string]interface{}{"@odata.id": "/redfish/v1/Managers/CMC.Integrated.1/AccountService"},
		})
	rh.createTreeCollectionLeaf(ctx, "/redfish/v1/Systems", map[string]interface{}{})
	rh.createTreeCollectionLeaf(ctx, "/redfish/v1/Chassis", map[string]interface{}{})
	rh.createTreeLeaf(ctx, "/redfish/v1/EventService",
		map[string]interface{}{
			"@odata.context":                        "/redfish/v1/$metadata#EventService",
			"@odata.id":                             "/redfish/v1/EventService",
			"@odata.type":                           "#EventService.v1_0_2.EventService",
			"EventTypesForSubscription@odata.count": 5,
			"Id":                           "EventService",
			"Name":                         "Event Service",
			"DeliveryRetryAttempts":        3,
			"DeliveryRetryIntervalSeconds": 30,
			"Description":                  "Event Service represents the properties for the service",
			"Subscriptions":                map[string]interface{}{"@odata.id": "/redfish/v1/EventService/Subscriptions"},
			"Status": map[string]interface{}{
				"Health":       "Ok",
				"HealthRollup": "Ok",
			},

			"Actions": map[string]interface{}{
				"#EventService.SubmitTestEvent": map[string]interface{}{
					"EventType@Redfish.AllowableValues": []string{
						"StatusChange",
						"ResourceUpdated",
						"ResourceAdded",
						"ResourceRemoved",
						"Alert",
					},
					"target": "/redfish/v1/EventService/Actions/EventService.SubmitTestEvent",
				},
			},

			"EventTypesForSubscription": []string{
				"StatusChange",
				"ResourceUpdated",
				"ResourceAdded",
				"ResourceRemoved",
				"Alert",
			},
		})
	rh.createTreeCollectionLeaf(ctx, "/redfish/v1/JSONSchemas", map[string]interface{}{})
	rh.createTreeCollectionLeaf(ctx, "/redfish/v1/Managers", map[string]interface{}{})
	rh.createTreeCollectionLeaf(ctx, "/redfish/v1/Registries", map[string]interface{}{})
}

func (rh *config) createTreeLeaf(ctx context.Context, uri string, Properties map[string]interface{}) {
	uuid := eh.NewUUID()
	rh.cmdbus.HandleCommand(ctx, &domain.CreateOdataResource{UUID: uuid, ResourceURI: uri, Properties: Properties})
}

func (rh *config) createTreeCollectionLeaf(ctx context.Context, uri string, Properties map[string]interface{}) {
	uuid := eh.NewUUID()
	rh.cmdbus.HandleCommand(ctx, &domain.CreateOdataResourceCollection{UUID: uuid, ResourceURI: uri, Properties: Properties, Members: map[string]string{}})
}