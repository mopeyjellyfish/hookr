package inspect

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/mopeyjellyfish/hookr/internal/devhost"
	"github.com/mopeyjellyfish/hookr/internal/schemautil"
	"github.com/mopeyjellyfish/hookr/internal/trustopts"
	hookrruntime "github.com/mopeyjellyfish/hookr/runtime"
)

type Config struct {
	SchemaPath        string
	PluginPath        string
	HostFixturePath   string
	Hash              string
	AllowUnsigned     bool
	FlatcPath         string
	IncludePaths      []string
	Package           string
	PluginService     string
	HostService       string
	OptionalAttribute string
	Stdout            io.Writer
	Stderr            io.Writer
}

func Run(cfg Config) error {
	if cfg.SchemaPath == "" {
		return errors.New("schema path is required")
	}
	runner, model, err := schemautil.Load(schemautil.Config{
		SchemaPath:        cfg.SchemaPath,
		FlatcPath:         cfg.FlatcPath,
		IncludePaths:      cfg.IncludePaths,
		Package:           cfg.Package,
		PluginService:     cfg.PluginService,
		HostService:       cfg.HostService,
		OptionalAttribute: cfg.OptionalAttribute,
	})
	if err != nil {
		return err
	}

	out := cfg.Stdout
	if out == nil {
		out = os.Stdout
	}
	errOut := cfg.Stderr
	if errOut == nil {
		errOut = os.Stderr
	}
	_, _ = fmt.Fprintf(errOut, "hookr: inspecting schema %s\n", cfg.SchemaPath)

	_, _ = fmt.Fprintf(out, "Contract: %s\n", model.Name)
	_, _ = fmt.Fprintf(out, "Schema: %s\n", model.SchemaPath)
	_, _ = fmt.Fprintf(out, "Package: %s\n", model.PackageName)
	_, _ = fmt.Fprintf(out, "Schema Hash: %s\n", hex.EncodeToString(model.SchemaHash[:]))
	_, _ = fmt.Fprintf(out, "Plugin Service: %s\n", model.PluginService.Name)
	for _, method := range model.PluginService.Methods {
		required := "required"
		if method.Optional {
			required = "optional"
		}
		_, _ = fmt.Fprintf(
			out,
			"  - %s id=%d req=%s resp=%s %s\n",
			method.Name,
			method.ID,
			method.RequestType,
			method.ResponseType,
			required,
		)
	}
	if model.HostService != nil {
		_, _ = fmt.Fprintf(out, "Host Service: %s\n", model.HostService.Name)
		for _, method := range model.HostService.Methods {
			_, _ = fmt.Fprintf(
				out,
				"  - %s id=%d req=%s resp=%s\n",
				method.Name,
				method.ID,
				method.RequestType,
				method.ResponseType,
			)
		}
	}

	if cfg.PluginPath == "" {
		return nil
	}
	_, _ = fmt.Fprintf(errOut, "hookr: loading plugin %s\n", cfg.PluginPath)

	fileOptions, err := trustopts.Build(cfg.Hash, cfg.AllowUnsigned)
	if err != nil {
		return err
	}
	hostFixture, err := devhost.LoadFixture(cfg.HostFixturePath)
	if err != nil {
		return err
	}
	opts := []hookrruntime.Option{
		hookrruntime.WithFile(cfg.PluginPath, fileOptions...),
	}
	hostMethods := devhost.BindHostMethods(
		model,
		runner,
		cfg.SchemaPath,
		cfg.IncludePaths,
		hostFixture,
	)
	if len(hostMethods) > 0 {
		opts = append(opts, hookrruntime.WithHostMethodFns(hostMethods...))
	}
	rt, err := hookrruntime.New(context.Background(), opts...)
	if err != nil {
		return err
	}
	defer func() {
		_ = rt.Close(context.Background())
	}()

	_, _ = fmt.Fprintf(out, "Plugin Artifact: %s\n", cfg.PluginPath)
	_, _ = fmt.Fprintf(out, "Method ABI: %t\n", rt.SupportsMethodABI())
	if hs, ok := rt.PluginHandshake(); ok {
		_, _ = fmt.Fprintf(out, "Plugin ABI: %d.%d\n", hs.ABIMajor, hs.ABIMinor)
		_, _ = fmt.Fprintf(out, "Plugin Schema Hash: %s\n", hex.EncodeToString(hs.SchemaHash[:]))
		_, _ = fmt.Fprintf(out, "Plugin Capabilities: 0x%x\n", hs.Capabilities)
		_, _ = fmt.Fprintf(out, "Schema Hash Match: %t\n", hs.SchemaHash == model.SchemaHash)
	}
	pluginMethods := rt.PluginMethodIDs()
	if len(pluginMethods) == 0 {
		_, _ = fmt.Fprintf(out, "Plugin Methods: none reported\n")
		return nil
	}
	_, _ = fmt.Fprintf(out, "Plugin Methods:\n")
	for _, methodID := range pluginMethods {
		name := "unknown"
		status := "extra"
		for _, method := range model.PluginService.Methods {
			if method.ID == methodID {
				name = method.Name
				status = "optional"
				if !method.Optional {
					status = "required"
				}
				break
			}
		}
		_, _ = fmt.Fprintf(out, "  - %s id=%d %s\n", name, methodID, status)
	}
	_, _ = fmt.Fprintf(out, "Contract Methods:\n")
	for _, method := range model.PluginService.Methods {
		state := "missing"
		if rt.HasPluginMethodID(method.ID) {
			state = "implemented"
		}
		required := "required"
		if method.Optional {
			required = "optional"
		}
		_, _ = fmt.Fprintf(out, "  - %s id=%d %s %s\n", method.Name, method.ID, required, state)
	}
	_, _ = fmt.Fprintf(errOut, "hookr: inspected plugin %s\n", cfg.PluginPath)
	return nil
}
