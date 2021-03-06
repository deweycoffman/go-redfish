package view

import (
	"github.com/superchalupa/go-redfish/src/ocp/model"
	domain "github.com/superchalupa/go-redfish/src/redfishresource"
)

func WithURI(name string) Option {
	return func(s *View) error {
		s.pluginType = domain.PluginType(name)
		s.viewURI = name
		return nil
	}
}

func WithModel(name string, m *model.Model) Option {
	return func(s *View) error {
		s.models[name] = m
		return nil
	}
}

func WithFormatter(name string, g formatter) Option {
	return func(s *View) error {
		s.outputFormatters[name] = g
		return nil
	}
}

func WithController(name string, c controller) Option {
	return func(s *View) error {
		s.controllers[name] = c
		return nil
	}
}

func WithAction(name string, a Action) Option {
	return func(s *View) error {
		s.actions[name] = a
		return nil
	}
}

func WatchModel(name string, fn func(*View, *model.Model, []model.Update)) Option {
	return func(s *View) error {
		if m, ok := s.models[name]; ok {
			m.AddObserver(s.GetURIUnlocked(), func(m *model.Model, updates []model.Update) {
				fn(s, m, updates)
			})
		}
		return nil
	}
}
