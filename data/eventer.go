package data

import (
	"github.com/fletaio/common"
	"github.com/fletaio/core/event"
)

// Eventer provide Event's handlers of the target chain
type Eventer struct {
	chainCoord     *common.Coordinate
	handlerTypeMap map[event.Type]*EventHandler
	typeNameMap    map[string]event.Type
	typeMap        map[event.Type]*EventTypeItem
}

// NewEventer returns a Eventer
func NewEventer(ChainCoord *common.Coordinate) *Eventer {
	act := &Eventer{
		chainCoord:     ChainCoord,
		handlerTypeMap: map[event.Type]*EventHandler{},
		typeNameMap:    map[string]event.Type{},
		typeMap:        map[event.Type]*EventTypeItem{},
	}
	return act
}

// ChainCoord returns the coordinate of the target chain
func (act *Eventer) ChainCoord() *common.Coordinate {
	return act.chainCoord
}

// RegisterType add the Event type with handler loaded by the name from the global Event registry
func (act *Eventer) RegisterType(Name string, t event.Type) error {
	item, err := loadEventHandler(Name)
	if err != nil {
		return err
	}
	act.typeMap[t] = &EventTypeItem{
		Type:    t,
		Name:    Name,
		Factory: item.Factory,
	}
	act.handlerTypeMap[t] = item
	act.typeNameMap[Name] = t
	return nil
}

// NewByType generate an Event instance by the type
func (act *Eventer) NewByType(t event.Type) (event.Event, error) {
	if item, has := act.typeMap[t]; has {
		acc := item.Factory(t)
		return acc, nil
	} else {
		return nil, ErrUnknownEventType
	}
}

// NewByTypeName generate an Event instance by the name
func (act *Eventer) NewByTypeName(name string) (event.Event, error) {
	if t, has := act.typeNameMap[name]; has {
		return act.NewByType(t)
	} else {
		return nil, ErrUnknownEventType
	}
}

// TypeByName returns the type by the name
func (act *Eventer) TypeByName(name string) (event.Type, error) {
	if t, has := act.typeNameMap[name]; has {
		return t, nil
	} else {
		return 0, ErrUnknownEventType
	}
}

// NameByType returns the name by the type
func (act *Eventer) NameByType(t event.Type) (string, error) {
	if item, has := act.typeMap[t]; has {
		return item.Name, nil
	} else {
		return "", ErrUnknownEventType
	}
}

var EventerHandlerMap = map[string]*EventHandler{}

// RegisterEvent register Event handlers to the global Event registry
func RegisterEvent(Name string, Factory EventFactory) error {
	if _, has := EventerHandlerMap[Name]; has {
		return ErrExistHandler
	}
	EventerHandlerMap[Name] = &EventHandler{
		Factory: Factory,
	}
	return nil
}

func loadEventHandler(Name string) (*EventHandler, error) {
	if _, has := EventerHandlerMap[Name]; !has {
		return nil, ErrNotExistHandler
	}
	return EventerHandlerMap[Name], nil
}

type EventHandler struct {
	Factory EventFactory
}

type EventTypeItem struct {
	Type    event.Type
	Name    string
	Factory EventFactory
}

// EventFactory is a function type to generate an Event instance by the type
type EventFactory func(t event.Type) event.Event
