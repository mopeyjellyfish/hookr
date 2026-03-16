package contract

import (
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mopeyjellyfish/hookr/internal/flatbuffers/reflection"
)

const (
	DefaultPluginService = "Plugin"
	OptionalAttribute    = "hookr_optional"
)

var (
	ErrPluginServiceMissing = errors.New("plugin service not found in contract")
	ErrMethodIDCollision    = errors.New("derived method id collision")
	ErrServiceNameCollision = errors.New("service short-name collision")
)

type LoadOptions struct {
	SchemaPath        string
	BFBSPath          string
	PackageName       string
	ContractName      string
	PluginServiceName string
	OptionalAttribute string
}

type Contract struct {
	Name          string
	PackageName   string
	SchemaPath    string
	BFBSPath      string
	PluginService Service
	HostServices  []Service
	SchemaHash    [32]byte
}

type Service struct {
	Name    string
	Methods []Method
}

type Method struct {
	ID                uint32
	ServiceName       string
	Name              string
	RequestType       string
	ResponseType      string
	RequestQualified  string
	ResponseQualified string
	Optional          bool
	Attributes        map[string]string
}

func (c Contract) PluginMethod(name string) (Method, bool) {
	return c.PluginService.Method(name)
}

func (c Contract) HostService(name string) (Service, bool) {
	for _, service := range c.HostServices {
		if service.Name == name {
			return service, true
		}
	}
	return Service{}, false
}

func (c Contract) HostMethod(serviceName string, methodName string) (Method, bool) {
	service, ok := c.HostService(serviceName)
	if !ok {
		return Method{}, false
	}
	return service.Method(methodName)
}

func (s Service) Method(name string) (Method, bool) {
	for _, method := range s.Methods {
		if method.Name == name {
			return method, true
		}
	}
	return Method{}, false
}

func Load(opts LoadOptions) (Contract, error) {
	if opts.BFBSPath == "" {
		return Contract{}, errors.New("bfbs path is required")
	}
	if opts.PackageName == "" {
		return Contract{}, errors.New("package name is required")
	}
	pluginServiceName := opts.PluginServiceName
	if pluginServiceName == "" {
		pluginServiceName = DefaultPluginService
	}
	optionalAttribute := opts.OptionalAttribute
	if optionalAttribute == "" {
		optionalAttribute = OptionalAttribute
	}

	data, err := os.ReadFile(opts.BFBSPath)
	if err != nil {
		return Contract{}, fmt.Errorf("read bfbs: %w", err)
	}
	schema := reflection.GetRootAsSchema(data, 0)

	pluginFullName, err := resolveServiceName(schema, pluginServiceName)
	if err != nil {
		return Contract{}, err
	}

	plugin, err := loadService(schema, pluginFullName, optionalAttribute)
	if err != nil {
		return Contract{}, err
	}
	hostServices, err := loadHostServices(schema, pluginFullName, optionalAttribute)
	if err != nil {
		return Contract{}, err
	}
	if err := ensureUniqueServiceNames(plugin, hostServices); err != nil {
		return Contract{}, err
	}
	if err := ensureUniqueMethodIDs(plugin, hostServices); err != nil {
		return Contract{}, err
	}

	contract := Contract{
		Name:          contractName(opts.ContractName, opts.SchemaPath, plugin.Name),
		PackageName:   opts.PackageName,
		SchemaPath:    opts.SchemaPath,
		BFBSPath:      opts.BFBSPath,
		PluginService: plugin,
		HostServices:  hostServices,
	}
	contract.SchemaHash = canonicalHash(schema, contract)
	return contract, nil
}

func loadHostServices(
	schema *reflection.Schema,
	pluginServiceName string,
	optionalAttr string,
) ([]Service, error) {
	services := make([]Service, 0, schema.ServicesLength())
	var svc reflection.Service
	for i := 0; i < schema.ServicesLength(); i++ {
		if !schema.Services(&svc, i) {
			continue
		}
		fullName := bytesToString(svc.Name())
		if fullName == pluginServiceName {
			continue
		}
		service, err := loadService(schema, fullName, optionalAttr)
		if err != nil {
			return nil, err
		}
		services = append(services, service)
	}
	sortServices(services)
	return services, nil
}

func loadService(schema *reflection.Schema, wanted string, optionalAttr string) (Service, error) {
	var svc reflection.Service
	for i := 0; i < schema.ServicesLength(); i++ {
		if !schema.Services(&svc, i) {
			continue
		}
		fullName := bytesToString(svc.Name())
		if fullName != wanted {
			continue
		}
		name := shortName(fullName)
		methods := make([]Method, 0, svc.CallsLength())
		var call reflection.RPCCall
		for j := 0; j < svc.CallsLength(); j++ {
			if !svc.Calls(&call, j) {
				continue
			}
			requestObj := call.Request(nil)
			responseObj := call.Response(nil)
			methodName := bytesToString(call.Name())
			method := Method{
				ID:                deriveMethodID(name, methodName),
				ServiceName:       name,
				Name:              methodName,
				RequestQualified:  bytesToString(requestObj.Name()),
				ResponseQualified: bytesToString(responseObj.Name()),
				RequestType:       shortName(bytesToString(requestObj.Name())),
				ResponseType:      shortName(bytesToString(responseObj.Name())),
				Attributes:        map[string]string{},
			}
			var attr reflection.KeyValue
			for k := 0; k < call.AttributesLength(); k++ {
				if !call.Attributes(&attr, k) {
					continue
				}
				key := bytesToString(attr.Key())
				value := bytesToString(attr.Value())
				method.Attributes[key] = value
				if key == optionalAttr {
					method.Optional = true
				}
			}
			methods = append(methods, method)
		}
		return Service{Name: name, Methods: methods}, nil
	}
	return Service{}, fmt.Errorf("%w: %s", ErrPluginServiceMissing, wanted)
}

func resolveServiceName(schema *reflection.Schema, wanted string) (string, error) {
	var svc reflection.Service
	shortMatches := make([]string, 0, 1)
	for i := 0; i < schema.ServicesLength(); i++ {
		if !schema.Services(&svc, i) {
			continue
		}
		fullName := bytesToString(svc.Name())
		if fullName == wanted {
			return fullName, nil
		}
		if shortName(fullName) == wanted {
			shortMatches = append(shortMatches, fullName)
		}
	}
	switch len(shortMatches) {
	case 0:
		return "", fmt.Errorf("%w: %s", ErrPluginServiceMissing, wanted)
	case 1:
		return shortMatches[0], nil
	default:
		return "", fmt.Errorf("plugin service %q is ambiguous: %s", wanted, strings.Join(shortMatches, ", "))
	}
}

func ensureUniqueServiceNames(plugin Service, hostServices []Service) error {
	seen := map[string]string{
		plugin.Name: plugin.Name,
	}
	for _, service := range hostServices {
		if prior, ok := seen[service.Name]; ok {
			return fmt.Errorf("%w: %s and %s", ErrServiceNameCollision, prior, service.Name)
		}
		seen[service.Name] = service.Name
	}
	return nil
}

func ensureUniqueMethodIDs(plugin Service, hostServices []Service) error {
	seen := map[uint32]string{}
	services := []Service{plugin}
	services = append(services, hostServices...)
	for _, service := range services {
		for _, method := range service.Methods {
			if prior, ok := seen[method.ID]; ok {
				return fmt.Errorf("%w: %s and %s", ErrMethodIDCollision, prior, methodKey(method))
			}
			seen[method.ID] = methodKey(method)
		}
	}
	return nil
}

func deriveMethodID(serviceName, methodName string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(serviceName))
	_, _ = h.Write([]byte{':'})
	_, _ = h.Write([]byte(methodName))
	return h.Sum32()
}

func bytesToString(data []byte) string {
	return string(data)
}

func shortName(name string) string {
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		return name[idx+1:]
	}
	return name
}

func contractName(override, schemaPath, pluginServiceName string) string {
	if override != "" {
		return override
	}
	if schemaPath != "" {
		base := strings.TrimSuffix(filepath.Base(schemaPath), filepath.Ext(schemaPath))
		if base != "" {
			return toExportedIdentifier(base)
		}
	}
	return toExportedIdentifier(pluginServiceName)
}

func methodKey(method Method) string {
	return method.ServiceName + "." + method.Name
}

func sortServices(services []Service) {
	sort.Slice(services, func(i, j int) bool { return services[i].Name < services[j].Name })
}

func toExportedIdentifier(s string) string {
	if s == "" {
		return "Unnamed"
	}
	var b strings.Builder
	upperNext := true
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
			if b.Len() == 0 && r >= '0' && r <= '9' {
				b.WriteByte('M')
			}
			if upperNext && r >= 'a' && r <= 'z' {
				r = r - ('a' - 'A')
			}
			b.WriteRune(r)
			upperNext = false
		default:
			upperNext = true
		}
	}
	if b.Len() == 0 {
		return "Unnamed"
	}
	return b.String()
}
