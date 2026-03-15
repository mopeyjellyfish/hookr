package call

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mopeyjellyfish/hookr/internal/flatbuffers/reflection"
)

func buildTemplateJSON(schema *reflection.Schema, qualifiedName string) ([]byte, error) {
	if schema == nil {
		return nil, errors.New("schema reflection is unavailable")
	}
	graph := newTemplateGraph(schema)
	value, err := graph.objectTemplate(qualifiedName)
	if err != nil {
		return nil, err
	}
	return json.Marshal(value)
}

type templateGraph struct {
	objectsByName map[string]*reflection.Object
	enumsByName   map[string]*reflection.Enum
	objectsByIdx  map[int32]*reflection.Object
	enumsByIdx    map[int32]*reflection.Enum
}

func newTemplateGraph(schema *reflection.Schema) *templateGraph {
	g := &templateGraph{
		objectsByName: map[string]*reflection.Object{},
		enumsByName:   map[string]*reflection.Enum{},
		objectsByIdx:  map[int32]*reflection.Object{},
		enumsByIdx:    map[int32]*reflection.Enum{},
	}
	for i := 0; i < schema.ObjectsLength(); i++ {
		obj := &reflection.Object{}
		if !schema.Objects(obj, i) {
			continue
		}
		name := string(obj.Name())
		g.objectsByName[name] = obj
		g.objectsByIdx[int32(i)] = obj
	}
	for i := 0; i < schema.EnumsLength(); i++ {
		enum := &reflection.Enum{}
		if !schema.Enums(enum, i) {
			continue
		}
		name := string(enum.Name())
		g.enumsByName[name] = enum
		g.enumsByIdx[int32(i)] = enum
	}
	return g
}

func (g *templateGraph) objectTemplate(name string) (map[string]any, error) {
	obj := g.objectsByName[name]
	if obj == nil {
		return nil, fmt.Errorf("object %s not found in schema", name)
	}
	out := map[string]any{}
	var field reflection.Field
	for i := 0; i < obj.FieldsLength(); i++ {
		if !obj.Fields(&field, i) {
			continue
		}
		value, err := g.typeTemplate(field.Type(nil), &field)
		if err != nil {
			return nil, fmt.Errorf("template for %s.%s: %w", name, string(field.Name()), err)
		}
		out[string(field.Name())] = value
	}
	return out, nil
}

func (g *templateGraph) typeTemplate(typ *reflection.Type, field *reflection.Field) (any, error) {
	if typ == nil {
		return nil, nil
	}
	switch typ.BaseType() {
	case reflection.BaseTypeBool:
		return field.DefaultInteger() != 0, nil
	case reflection.BaseTypeByte,
		reflection.BaseTypeShort,
		reflection.BaseTypeInt,
		reflection.BaseTypeLong:
		return field.DefaultInteger(), nil
	case reflection.BaseTypeUByte,
		reflection.BaseTypeUShort,
		reflection.BaseTypeUInt,
		reflection.BaseTypeULong,
		reflection.BaseTypeUType:
		return uint64(field.DefaultInteger()), nil
	case reflection.BaseTypeFloat, reflection.BaseTypeDouble:
		return field.DefaultReal(), nil
	case reflection.BaseTypeString:
		return "", nil
	case reflection.BaseTypeVector, reflection.BaseTypeVector64, reflection.BaseTypeArray:
		if typ.Element() == reflection.BaseTypeObj {
			refName := g.referenceName(typ)
			if refName != "" {
				obj, err := g.objectTemplate(refName)
				if err != nil {
					return nil, err
				}
				return []any{obj}, nil
			}
		}
		return []any{}, nil
	case reflection.BaseTypeObj:
		refName := g.referenceName(typ)
		if refName == "" {
			return map[string]any{}, nil
		}
		return g.objectTemplate(refName)
	default:
		if refName := g.referenceName(typ); refName != "" {
			if enum := g.enumsByName[refName]; enum != nil && enum.ValuesLength() > 0 {
				var enumVal reflection.EnumVal
				if enum.Values(&enumVal, 0) {
					return string(enumVal.Name()), nil
				}
			}
		}
		return nil, nil
	}
}

func (g *templateGraph) referenceName(typ *reflection.Type) string {
	if typ == nil || typ.Index() < 0 {
		return ""
	}
	index := typ.Index()
	switch typ.BaseType() {
	case reflection.BaseTypeObj:
		if obj := g.objectsByIdx[index]; obj != nil {
			return string(obj.Name())
		}
	case reflection.BaseTypeVector, reflection.BaseTypeVector64, reflection.BaseTypeArray:
		if typ.Element() == reflection.BaseTypeObj {
			if obj := g.objectsByIdx[index]; obj != nil {
				return string(obj.Name())
			}
		}
	default:
		if enum := g.enumsByIdx[index]; enum != nil {
			return string(enum.Name())
		}
	}
	return ""
}
