package key

// Type TODO
type Type uint8

// String TODO
func (t Type) String() string {
	if item, has := typeHash[t]; has {
		return item.Name
	} else {
		return "Unknown"
	}
}

// Factory TODO
type Factory func() Key

type typeItem struct {
	Type    Type
	Name    string
	Factory Factory
}

var typeHash = map[Type]*typeItem{}

// AddType TODO
func AddType(Type Type, Name string, Factory Factory) {
	typeHash[Type] = &typeItem{
		Type:    Type,
		Name:    Name,
		Factory: Factory,
	}
}

// NewByType TODO
func NewByType(t Type) (Key, error) {
	if item, has := typeHash[t]; has {
		return item.Factory(), nil
	} else {
		return nil, ErrUnknownKeyType
	}
}
