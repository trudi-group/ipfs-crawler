package crawling

import (
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"gopkg.in/yaml.v3"
)

var (
	driversM sync.RWMutex
	drivers  = make(map[string]Driver)

	// ErrPluginDoesNotExist is the error returned by NewPlugin when a
	// plugin with that name does not exist.
	ErrPluginDoesNotExist = fmt.Errorf("middleware driver with that name does not exist")
)

// A Driver is a provider for a plugin implementation.
type Driver interface {
	// NewImpl creates a new implementation of the plugin.
	// It is provided with a reference to the libp2p host it runs in, and
	// a yaml-encoded representation of its configuration.
	NewImpl(h host.Host, cfg []byte) (Plugin, error)
}

// RegisterPlugin makes a Driver available by the provided name.
//
// If called twice with the same name, the name is blank, or if the provided
// Driver is nil, this function panics.
func RegisterPlugin(name string, d Driver) {
	if name == "" {
		panic("plugin: could not register a Driver with an empty name")
	}
	if d == nil {
		panic("plugin: could not register a nil Driver")
	}

	driversM.Lock()
	defer driversM.Unlock()

	if _, dup := drivers[name]; dup {
		panic("plugin: RegisterPlugin called twice for " + name)
	}

	drivers[name] = d
}

// A Plugin exposes functionality to measure peers encountered during a crawl.
type Plugin interface {
	// Name returns the name of the plugin.
	Name() string

	// HandlePeer measures the given peer.
	// The underlying libp2p node should have an open connection to the peer.
	// The success value returned must be serializable to JSON and will be
	// copied verbose into the crawl output.
	// TODO maybe this only needs peer ID? Or network.Conn?
	HandlePeer(info peer.AddrInfo) (interface{}, error)

	// Shutdown ensures clean shutdown of this plugin.
	Shutdown() error
}

// NewPlugin attempts to initialize a new plugin instance from the
// list of registered Plugins.
//
// If a plugin does not exist, returns ErrPluginDoesNotExist.
func NewPlugin(name string, h host.Host, optionBytes []byte) (Plugin, error) {
	driversM.RLock()
	defer driversM.RUnlock()

	var d Driver
	d, ok := drivers[name]
	if !ok {
		return nil, ErrPluginDoesNotExist
	}

	return d.NewImpl(h, optionBytes)
}

// PluginConfig is the generic configuration format used for all registered Plugins.
type PluginConfig struct {
	Name    string                 `yaml:"name"`
	Options map[string]interface{} `yaml:"options"`
}

// PluginsFromPluginConfigs is a utility function for initializing Plugins in bulk.
func PluginsFromPluginConfigs(h host.Host, cfgs []PluginConfig) ([]Plugin, error) {
	var plugins []Plugin

	for _, cfg := range cfgs {
		// Marshal the options back into bytes.
		var optionBytes []byte
		optionBytes, err := yaml.Marshal(cfg.Options)
		if err != nil {
			return nil, err
		}

		var p Plugin
		p, err = NewPlugin(cfg.Name, h, optionBytes)
		if err != nil {
			return nil, err
		}

		plugins = append(plugins, p)
	}

	return plugins, nil
}
