# Hookr WASM RT

This is a runtime for the Hookr WASM plugin system. It is a standalone runtime that is injected with other modules to provide a common interface between the host and the wasm code, implementing memory functions and other utilities for reading and writing data out of the plugin.

## Build

To build the runtime, run the following command:

```sh
make wasm/runtime
```
