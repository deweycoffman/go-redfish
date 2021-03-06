package view

import (
	"context"
	"sync"

	eh "github.com/looplab/eventhorizon"
	"github.com/superchalupa/go-redfish/src/ocp/model"
	domain "github.com/superchalupa/go-redfish/src/redfishresource"
)

type Option func(*View) error

type controller interface {
	UpdateRequest(ctx context.Context, property string, value interface{}) (interface{}, error)
}

type formatter func(
	ctx context.Context,
	v *View,
	m *model.Model,
	rrp *domain.RedfishResourceProperty,
	meta map[string]interface{},
) error

type Action func(context.Context, eh.Event, *domain.HTTPCmdProcessedData) error

type View struct {
	sync.RWMutex
	pluginType       domain.PluginType
	viewURI          string
	uuid             eh.UUID
	controllers      map[string]controller
	models           map[string]*model.Model
	outputFormatters map[string]formatter
	actions          map[string]Action
	actionURI        map[string]string
}

func New(options ...Option) *View {
	s := &View{
		uuid:             eh.NewUUID(),
		controllers:      map[string]controller{},
		models:           map[string]*model.Model{},
		outputFormatters: map[string]formatter{},
		actions:          map[string]Action{},
		actionURI:        map[string]string{},
	}

	s.ApplyOption(options...)
	domain.RegisterPlugin(func() domain.Plugin { return s })
	return s
}

func (s *View) Close() {
	domain.UnregisterPlugin(s.pluginType)
}

func (s *View) ApplyOption(options ...Option) error {
	s.Lock()
	defer s.Unlock()
	for _, o := range options {
		err := o(s)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *View) PluginType() domain.PluginType {
	return s.pluginType
}

func (s *View) GetUUID() eh.UUID {
	s.RLock()
	defer s.RUnlock()
	return s.uuid
}

func (s *View) GetURIUnlocked() string {
	return s.viewURI
}

func (s *View) GetURI() string {
	s.RLock()
	defer s.RUnlock()
	return s.GetURIUnlocked()
}

func (s *View) GetModel(name string) *model.Model {
	s.RLock()
	defer s.RUnlock()
	return s.models[name]
}

func (s *View) GetController(name string) controller {
	s.RLock()
	defer s.RUnlock()
	return s.controllers[name]
}

func (s *View) GetAction(name string) Action {
	s.RLock()
	defer s.RUnlock()
	return s.actions[name]
}

func (s *View) GetActionURI(name string) string {
	s.RLock()
	defer s.RUnlock()
	return s.actionURI[name]
}

func (v *View) SetActionURIUnlocked(name string, URI string) {
	v.actionURI[name] = URI
}

func (v *View) SetActionUnlocked(name string, a Action) {
	v.actions[name] = a
}
