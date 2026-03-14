package contract

import (
	"crypto/sha256"
	"encoding/json"
	"sort"

	"github.com/mopeyjellyfish/hookr/internal/flatbuffers/reflection"
)

type canonicalMethod struct {
	ID                uint32            `json:"id"`
	Name              string            `json:"name"`
	RequestQualified  string            `json:"request"`
	ResponseQualified string            `json:"response"`
	Optional          bool              `json:"optional,omitempty"`
	Attributes        map[string]string `json:"attributes,omitempty"`
}

type canonicalService struct {
	Name    string            `json:"name"`
	Methods []canonicalMethod `json:"methods"`
}

type canonicalType struct {
	BaseType    string `json:"base_type"`
	Element     string `json:"element,omitempty"`
	Index       int32  `json:"index,omitempty"`
	FixedLength uint16 `json:"fixed_length,omitempty"`
	BaseSize    uint32 `json:"base_size,omitempty"`
	ElementSize uint32 `json:"element_size,omitempty"`
	Ref         string `json:"ref,omitempty"`
}

type canonicalField struct {
	Name           string            `json:"name"`
	ID             uint16            `json:"id"`
	Type           canonicalType     `json:"type"`
	DefaultInteger int64             `json:"default_integer,omitempty"`
	DefaultReal    float64           `json:"default_real,omitempty"`
	Required       bool              `json:"required,omitempty"`
	Optional       bool              `json:"optional,omitempty"`
	Deprecated     bool              `json:"deprecated,omitempty"`
	Key            bool              `json:"key,omitempty"`
	Padding        uint16            `json:"padding,omitempty"`
	Offset64       bool              `json:"offset64,omitempty"`
	Attributes     map[string]string `json:"attributes,omitempty"`
}

type canonicalObject struct {
	Name       string            `json:"name"`
	IsStruct   bool              `json:"is_struct,omitempty"`
	MinAlign   int32             `json:"min_align,omitempty"`
	ByteSize   int32             `json:"byte_size,omitempty"`
	Fields     []canonicalField  `json:"fields"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

type canonicalEnumValue struct {
	Name       string            `json:"name"`
	Value      int64             `json:"value"`
	UnionType  *canonicalType    `json:"union_type,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

type canonicalEnum struct {
	Name           string               `json:"name"`
	IsUnion        bool                 `json:"is_union,omitempty"`
	UnderlyingType *canonicalType       `json:"underlying_type,omitempty"`
	Values         []canonicalEnumValue `json:"values"`
	Attributes     map[string]string    `json:"attributes,omitempty"`
}

type canonicalContract struct {
	ABIVersion       string            `json:"abi_version"`
	FileIdentifier   string            `json:"file_identifier,omitempty"`
	FileExtension    string            `json:"file_extension,omitempty"`
	AdvancedFeatures uint64            `json:"advanced_features,omitempty"`
	Plugin           canonicalService  `json:"plugin"`
	Host             *canonicalService `json:"host,omitempty"`
	Objects          []canonicalObject `json:"objects,omitempty"`
	Enums            []canonicalEnum   `json:"enums,omitempty"`
}

func canonicalHash(schema *reflection.Schema, contract Contract) [32]byte {
	builder := newCanonicalGraph(schema)

	canonical := canonicalContract{
		ABIVersion:       "hookr-flatbuffers",
		FileIdentifier:   bytesToString(schema.FileIdent()),
		FileExtension:    bytesToString(schema.FileExt()),
		AdvancedFeatures: uint64(schema.AdvancedFeatures()),
		Plugin:           builder.service(contract.PluginService),
	}
	if contract.HostService != nil {
		host := builder.service(*contract.HostService)
		canonical.Host = &host
	}

	for _, method := range contract.PluginService.Methods {
		builder.visitObject(method.RequestQualified)
		builder.visitObject(method.ResponseQualified)
	}
	if contract.HostService != nil {
		for _, method := range contract.HostService.Methods {
			builder.visitObject(method.RequestQualified)
			builder.visitObject(method.ResponseQualified)
		}
	}

	canonical.Objects = builder.objects()
	canonical.Enums = builder.enums()

	payload, _ := json.Marshal(canonical)
	return sha256.Sum256(payload)
}

type canonicalGraph struct {
	schema        *reflection.Schema
	objectsByName map[string]*reflection.Object
	enumsByName   map[string]*reflection.Enum
	objectsByIdx  map[int32]*reflection.Object
	enumsByIdx    map[int32]*reflection.Enum
	visitedObj    map[string]struct{}
	visitedEnum   map[string]struct{}
	objectSet     map[string]canonicalObject
	enumSet       map[string]canonicalEnum
}

func newCanonicalGraph(schema *reflection.Schema) *canonicalGraph {
	g := &canonicalGraph{
		schema:        schema,
		objectsByName: map[string]*reflection.Object{},
		enumsByName:   map[string]*reflection.Enum{},
		objectsByIdx:  map[int32]*reflection.Object{},
		enumsByIdx:    map[int32]*reflection.Enum{},
		visitedObj:    map[string]struct{}{},
		visitedEnum:   map[string]struct{}{},
		objectSet:     map[string]canonicalObject{},
		enumSet:       map[string]canonicalEnum{},
	}

	for i := 0; i < schema.ObjectsLength(); i++ {
		obj := &reflection.Object{}
		if !schema.Objects(obj, i) {
			continue
		}
		name := bytesToString(obj.Name())
		g.objectsByName[name] = obj
		g.objectsByIdx[int32(i)] = obj
	}
	for i := 0; i < schema.EnumsLength(); i++ {
		enum := &reflection.Enum{}
		if !schema.Enums(enum, i) {
			continue
		}
		name := bytesToString(enum.Name())
		g.enumsByName[name] = enum
		g.enumsByIdx[int32(i)] = enum
	}
	return g
}

func (g *canonicalGraph) service(service Service) canonicalService {
	methods := make([]canonicalMethod, 0, len(service.Methods))
	for _, method := range service.Methods {
		methods = append(methods, canonicalMethod{
			ID:                method.ID,
			Name:              method.Name,
			RequestQualified:  method.RequestQualified,
			ResponseQualified: method.ResponseQualified,
			Optional:          method.Optional,
			Attributes:        sortedAttributes(method.Attributes),
		})
	}
	sort.Slice(methods, func(i, j int) bool { return methods[i].Name < methods[j].Name })
	return canonicalService{
		Name:    service.Name,
		Methods: methods,
	}
}

func (g *canonicalGraph) visitObject(name string) {
	if name == "" {
		return
	}
	if _, ok := g.visitedObj[name]; ok {
		return
	}
	obj := g.objectsByName[name]
	if obj == nil {
		return
	}
	g.visitedObj[name] = struct{}{}

	fields := make([]canonicalField, 0, obj.FieldsLength())
	var field reflection.Field
	for i := 0; i < obj.FieldsLength(); i++ {
		if !obj.Fields(&field, i) {
			continue
		}
		fieldType := field.Type(nil)
		canonicalFieldType := g.typeRef(fieldType)
		fields = append(fields, canonicalField{
			Name:           bytesToString(field.Name()),
			ID:             field.Id(),
			Type:           canonicalFieldType,
			DefaultInteger: field.DefaultInteger(),
			DefaultReal:    field.DefaultReal(),
			Required:       field.Required(),
			Optional:       field.Optional(),
			Deprecated:     field.Deprecated(),
			Key:            field.Key(),
			Padding:        field.Padding(),
			Offset64:       field.Offset64(),
			Attributes:     attributesFromField(&field),
		})
	}
	sort.Slice(fields, func(i, j int) bool {
		if fields[i].ID == fields[j].ID {
			return fields[i].Name < fields[j].Name
		}
		return fields[i].ID < fields[j].ID
	})

	g.objectSet[name] = canonicalObject{
		Name:       name,
		IsStruct:   obj.IsStruct(),
		MinAlign:   obj.Minalign(),
		ByteSize:   obj.Bytesize(),
		Fields:     fields,
		Attributes: attributesFromObject(obj),
	}
}

func (g *canonicalGraph) visitEnum(name string) {
	if name == "" {
		return
	}
	if _, ok := g.visitedEnum[name]; ok {
		return
	}
	enum := g.enumsByName[name]
	if enum == nil {
		return
	}
	g.visitedEnum[name] = struct{}{}

	values := make([]canonicalEnumValue, 0, enum.ValuesLength())
	var enumVal reflection.EnumVal
	for i := 0; i < enum.ValuesLength(); i++ {
		if !enum.Values(&enumVal, i) {
			continue
		}
		var unionType *canonicalType
		if raw := enumVal.UnionType(nil); raw != nil {
			ref := g.typeRef(raw)
			unionType = &ref
		}
		values = append(values, canonicalEnumValue{
			Name:       bytesToString(enumVal.Name()),
			Value:      enumVal.Value(),
			UnionType:  unionType,
			Attributes: attributesFromEnumVal(&enumVal),
		})
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Value == values[j].Value {
			return values[i].Name < values[j].Name
		}
		return values[i].Value < values[j].Value
	})

	var underlying *canonicalType
	if raw := enum.UnderlyingType(nil); raw != nil {
		ref := g.typeRef(raw)
		underlying = &ref
	}
	g.enumSet[name] = canonicalEnum{
		Name:           name,
		IsUnion:        enum.IsUnion(),
		UnderlyingType: underlying,
		Values:         values,
		Attributes:     attributesFromEnum(enum),
	}
}

func (g *canonicalGraph) typeRef(typ *reflection.Type) canonicalType {
	if typ == nil {
		return canonicalType{}
	}
	ref := canonicalType{
		BaseType:    typ.BaseType().String(),
		Element:     typ.Element().String(),
		Index:       typ.Index(),
		FixedLength: typ.FixedLength(),
		BaseSize:    typ.BaseSize(),
		ElementSize: typ.ElementSize(),
	}
	ref.Ref = g.typeReferenceName(typ)
	return ref
}

func (g *canonicalGraph) typeReferenceName(typ *reflection.Type) string {
	if typ == nil || typ.Index() < 0 {
		return ""
	}
	index := typ.Index()
	switch typ.BaseType() {
	case reflection.BaseTypeObj:
		if obj := g.objectsByIdx[index]; obj != nil {
			name := bytesToString(obj.Name())
			g.visitObject(name)
			return name
		}
	case reflection.BaseTypeUnion, reflection.BaseTypeUType:
		if enum := g.enumsByIdx[index]; enum != nil {
			name := bytesToString(enum.Name())
			g.visitEnum(name)
			return name
		}
	case reflection.BaseTypeVector, reflection.BaseTypeVector64, reflection.BaseTypeArray:
		switch typ.Element() {
		case reflection.BaseTypeObj:
			if obj := g.objectsByIdx[index]; obj != nil {
				name := bytesToString(obj.Name())
				g.visitObject(name)
				return name
			}
		case reflection.BaseTypeUnion, reflection.BaseTypeUType:
			if enum := g.enumsByIdx[index]; enum != nil {
				name := bytesToString(enum.Name())
				g.visitEnum(name)
				return name
			}
		}
	default:
		if enum := g.enumsByIdx[index]; enum != nil {
			name := bytesToString(enum.Name())
			g.visitEnum(name)
			return name
		}
	}
	return ""
}

func (g *canonicalGraph) objects() []canonicalObject {
	names := make([]string, 0, len(g.objectSet))
	for name := range g.objectSet {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]canonicalObject, 0, len(names))
	for _, name := range names {
		out = append(out, g.objectSet[name])
	}
	return out
}

func (g *canonicalGraph) enums() []canonicalEnum {
	names := make([]string, 0, len(g.enumSet))
	for name := range g.enumSet {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]canonicalEnum, 0, len(names))
	for _, name := range names {
		out = append(out, g.enumSet[name])
	}
	return out
}

func attributesFromObject(obj *reflection.Object) map[string]string {
	attrs := map[string]string{}
	var attr reflection.KeyValue
	for i := 0; i < obj.AttributesLength(); i++ {
		if !obj.Attributes(&attr, i) {
			continue
		}
		attrs[bytesToString(attr.Key())] = bytesToString(attr.Value())
	}
	return sortedAttributes(attrs)
}

func attributesFromField(field *reflection.Field) map[string]string {
	attrs := map[string]string{}
	var attr reflection.KeyValue
	for i := 0; i < field.AttributesLength(); i++ {
		if !field.Attributes(&attr, i) {
			continue
		}
		attrs[bytesToString(attr.Key())] = bytesToString(attr.Value())
	}
	return sortedAttributes(attrs)
}

func attributesFromEnum(enum *reflection.Enum) map[string]string {
	attrs := map[string]string{}
	var attr reflection.KeyValue
	for i := 0; i < enum.AttributesLength(); i++ {
		if !enum.Attributes(&attr, i) {
			continue
		}
		attrs[bytesToString(attr.Key())] = bytesToString(attr.Value())
	}
	return sortedAttributes(attrs)
}

func attributesFromEnumVal(enumVal *reflection.EnumVal) map[string]string {
	attrs := map[string]string{}
	var attr reflection.KeyValue
	for i := 0; i < enumVal.AttributesLength(); i++ {
		if !enumVal.Attributes(&attr, i) {
			continue
		}
		attrs[bytesToString(attr.Key())] = bytesToString(attr.Value())
	}
	return sortedAttributes(attrs)
}

func sortedAttributes(attrs map[string]string) map[string]string {
	if len(attrs) == 0 {
		return nil
	}
	keys := make([]string, 0, len(attrs))
	for key := range attrs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		out[key] = attrs[key]
	}
	return out
}
