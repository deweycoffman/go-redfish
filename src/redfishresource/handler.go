package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"sync"
	"time"

	"github.com/gorilla/mux"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/aggregatestore/model"
	"github.com/looplab/eventhorizon/commandhandler/aggregate"
	eventbus "github.com/looplab/eventhorizon/eventbus/local"
	eventpublisher "github.com/looplab/eventhorizon/publisher/local"
	repo "github.com/looplab/eventhorizon/repo/memory"

	"github.com/superchalupa/go-redfish/src/eventwaiter"
)

type waiter interface {
	Listen(context.Context, func(eh.Event) bool) (*eventwaiter.EventListener, error)
	Notify(context.Context, eh.Event)
}

type DomainObjects struct {
	CommandHandler eh.CommandHandler
	Repo           eh.ReadWriteRepo
	EventBus       eh.EventBus
	EventWaiter    waiter
	AggregateStore eh.AggregateStore
	EventPublisher eh.EventPublisher

	// for http returns
	HTTPResultsBus eh.EventBus
	HTTPPublisher  eh.EventPublisher
	HTTPWaiter     waiter

	treeMu sync.RWMutex
	Tree   map[string]eh.UUID

	collectionsMu sync.RWMutex
	collections   []string
}

// SetupDDDFunctions sets up the full Event Horizon domain
// returns a handler exposing some of the components.
func NewDomainObjects() (*DomainObjects, error) {
	d := DomainObjects{}

	d.Tree = make(map[string]eh.UUID)

	// Create the repository and wrap in a version repository.
	d.Repo = repo.NewRepo()

	// Create the event bus that distributes events.
	d.EventBus = eventbus.NewEventBus()
	d.EventPublisher = eventpublisher.NewEventPublisher()
	d.EventBus.AddHandler(eh.MatchAny(), d.EventPublisher)

	d.EventWaiter = eventwaiter.NewEventWaiter(eventwaiter.SetName("Main"))
	d.EventPublisher.AddObserver(d.EventWaiter)

	// specific event bus to handle returns from http
	d.HTTPResultsBus = eventbus.NewEventBus()
	d.HTTPPublisher = eventpublisher.NewEventPublisher()
	d.HTTPResultsBus.AddHandler(eh.MatchEvent(HTTPCmdProcessed), d.HTTPPublisher)

	// hook up http waiter to the other bus for back compat
	d.HTTPWaiter = eventwaiter.NewEventWaiter(eventwaiter.SetName("HTTP"))
	d.EventPublisher.AddObserver(d.HTTPWaiter)
	d.HTTPPublisher.AddObserver(d.HTTPWaiter)

	// set up commands so that they can directly publish to http bus
	eh.RegisterCommand(func() eh.Command {
		return &GET{
			HTTPEventBus: d.HTTPResultsBus,
		}
	})

	// set up our built-in observer
	d.EventPublisher.AddObserver(&d)

	// Create the aggregate repository.
	var err error
	d.AggregateStore, err = model.NewAggregateStore(d.Repo, d.EventBus)
	if err != nil {
		return nil, fmt.Errorf("could not create aggregate store: %s", err)
	}

	// Create the aggregate command handler.
	d.CommandHandler, err = aggregate.NewCommandHandler(AggregateType, d.AggregateStore)
	if err != nil {
		return nil, fmt.Errorf("could not create command handler: %s", err)
	}

	return &d, nil
}

func (d *DomainObjects) HasAggregateID(uri string) bool {
	d.treeMu.RLock()
	defer d.treeMu.RUnlock()
	_, ok := d.Tree[uri]
	return ok
}

func (d *DomainObjects) GetAggregateID(uri string) eh.UUID {
	d.treeMu.RLock()
	defer d.treeMu.RUnlock()
	return d.Tree[uri]
}

func (d *DomainObjects) GetAggregateIDOK(uri string) (id eh.UUID, ok bool) {
	d.treeMu.RLock()
	defer d.treeMu.RUnlock()
	id, ok = d.Tree[uri]
	return
}

func (d *DomainObjects) SetAggregateID(uri string, ID eh.UUID) {
	d.treeMu.Lock()
	defer d.treeMu.Unlock()
	d.Tree[uri] = ID
}

func (d *DomainObjects) DeleteResource(ctx context.Context, uri string) {
	d.treeMu.Lock()
	defer d.treeMu.Unlock()
	if UUID, ok := d.Tree[uri]; ok {
		as, ok := d.AggregateStore.(eh.WriteRepo)
		if ok {
			as.Remove(ctx, UUID)
		}
	}
	delete(d.Tree, uri)
}

// Notify implements the Notify method of the EventObserver interface.
func (d *DomainObjects) Notify(ctx context.Context, event eh.Event) {
	logger := ContextLogger(ctx, "domain")
	logger.Debug("Processing event", "event", event)
	if event.EventType() == RedfishResourceCreated {
		if data, ok := event.Data().(*RedfishResourceCreatedData); ok {
			// TODO: handle conflicts (how?)
			d.SetAggregateID(data.ResourceURI, data.ID)

			// TODO: need to split out auto collection management into a plugin
			if data.Collection {
				logger.Debug("New collection", "collection_name", data.ResourceURI)
				d.collectionsMu.Lock()
				d.collections = append(d.collections, data.ResourceURI)
				d.collectionsMu.Unlock()
			}

			collectionToTest := path.Dir(data.ResourceURI)
			d.collectionsMu.RLock()
			for _, v := range d.collections {
				if v == collectionToTest {
					logger.Debug("Add member to collection", "collection_name", v, "new_uri", data.ResourceURI)
					d.CommandHandler.HandleCommand(
						ctx,
						&AddResourceToRedfishResourceCollection{
							ID:          d.GetAggregateID(collectionToTest),
							ResourceURI: data.ResourceURI,
						},
					)
				}
			}
			d.collectionsMu.RUnlock()
		}
		return
	}
	if event.EventType() == RedfishResourceRemoved {
		if data, ok := event.Data().(*RedfishResourceRemovedData); ok {
			// Look to see if it is a member of a collection
			logger.Debug("Delete", "event", event)
			collectionToTest := path.Dir(data.ResourceURI)
			d.collectionsMu.RLock()
			for _, v := range d.collections {
				if v == collectionToTest {
					logger.Debug("Remove member from collection", "collection_name", v, "new_uri", data.ResourceURI)
					d.CommandHandler.HandleCommand(
						ctx,
						&RemoveResourceFromRedfishResourceCollection{
							ID:          d.GetAggregateID(collectionToTest),
							ResourceURI: data.ResourceURI,
						},
					)
				}
			}
			d.collectionsMu.RUnlock()

			// is this a collection? If so, remove it from our collections list
			d.collectionsMu.Lock()
			for i, c := range d.collections {
				if c == data.ResourceURI {
					logger.Debug("Remove collection", "collection_name", data.ResourceURI)
					// swap the collection we found to the end
					d.collections[len(d.collections)-1], d.collections[i] = d.collections[i], d.collections[len(d.collections)-1]
					// then slice it off
					d.collections = d.collections[:len(d.collections)-1]
					break
				}
			}
			d.collectionsMu.Unlock()

			// TODO: remove from aggregatestore?
			logger.Debug("Delete Resource", "URI", data.ResourceURI)
			d.DeleteResource(ctx, data.ResourceURI)
		}
		return
	}
}

// CommandHandler is a HTTP handler for eventhorizon.Commands. Commands must be
// registered with eventhorizon.RegisterCommand(). It expects a POST with a JSON
// body that will be unmarshalled into the command.
func (d *DomainObjects) GetInternalCommandHandler(backgroundCtx context.Context) http.Handler {
	m := sync.Mutex{}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		m.Lock()
		defer m.Unlock()

		if r.Method != "POST" {
			http.Error(w, "unsuported method: "+r.Method, http.StatusMethodNotAllowed)
			return
		}

		cmd, err := eh.CreateCommand(eh.CommandType("internal:" + vars["command"]))
		if err != nil {
			http.Error(w, "could not create command: "+err.Error(), http.StatusBadRequest)
			return
		}

		b, err := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			http.Error(w, "could not read command: "+err.Error(), http.StatusBadRequest)
			return
		}
		r.Body.Close()

		if err := json.Unmarshal(b, &cmd); err != nil {
			http.Error(w, "could not decode command: "+err.Error(), http.StatusBadRequest)
			return
		}

		// to avoid races, set up our listener first
		cmdID := eh.NewUUID()
		reqCtx := WithRequestID(r.Context(), cmdID)
		l, err := d.HTTPWaiter.Listen(reqCtx, func(event eh.Event) bool {
			if event.EventType() != HTTPCmdProcessed {
				return false
			}
			if data, ok := event.Data().(*HTTPCmdProcessedData); ok {
				if data.CommandID == cmdID {
					return true
				}
			}
			return false
		})
		if err != nil {
			http.Error(w, "could not create waiter"+err.Error(), http.StatusInternalServerError)
			return
		}
		l.Name = "Redfish HTTP Listener"
		defer l.Close()

		// NOTE: Use a new context when handling, else it will be cancelled with
		// the HTTP request which will cause projectors etc to fail if they run
		// async in goroutines past the request.
		if err := d.CommandHandler.HandleCommand(backgroundCtx, cmd); err != nil {
			http.Error(w, "could not handle command: "+err.Error(), http.StatusBadRequest)
			return
		}

		// send ourselves a message and wait for it to clear out the pipes
		data := &HTTPCmdProcessedData{
			CommandID:  cmdID,
			StatusCode: 200,
		}
		d.EventBus.PublishEvent(reqCtx, eh.NewEvent(HTTPCmdProcessed, data, time.Now()))

		_, _ = l.Wait(reqCtx)

		w.WriteHeader(http.StatusOK)
	})
}
