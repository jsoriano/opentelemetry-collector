// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/confmap"
)

// ConfigProvider provides the service configuration.
//
// The typical usage is the following:
//
//	cfgProvider.Get(...)
//	cfgProvider.Watch() // wait for an event.
//	cfgProvider.Get(...)
//	cfgProvider.Watch() // wait for an event.
//	// repeat Get/Watch cycle until it is time to shut down the Collector process.
//	cfgProvider.Shutdown()
type ConfigProvider interface {
	// Get returns the service configuration, or error otherwise.
	//
	// Should never be called concurrently with itself, Watch or Shutdown.
	Get(ctx context.Context, factories Factories) (*Config, error)

	// Watch blocks until any configuration change was detected or an unrecoverable error
	// happened during monitoring the configuration changes.
	//
	// Error is nil if the configuration is changed and needs to be re-fetched. Any non-nil
	// error indicates that there was a problem with watching the config changes.
	//
	// Should never be called concurrently with itself or Get.
	Watch() <-chan error

	// Shutdown signals that the provider is no longer in use and the that should close
	// and release any resources that it may have created.
	//
	// This function must terminate the Watch channel.
	//
	// Should never be called concurrently with itself or Get.
	Shutdown(ctx context.Context) error
}

type configProvider struct {
	mapResolver *confmap.Resolver
	processors  []ConfigProcessor
}

var _ ConfigProvider = (*configProvider)(nil)

// ConfigProviderSettings are the settings to configure the behavior of the ConfigProvider.
type ConfigProviderSettings struct {
	// ResolverSettings are the settings to configure the behavior of the confmap.Resolver.
	ResolverSettings confmap.ResolverSettings

	// ConfigProcessors are the configuration processors included.
	ConfigProcessors []ConfigProcessor
}

// NewConfigProvider returns a new ConfigProvider that provides the service configuration:
// * Initially it resolves the "configuration map":
//   - Retrieve the confmap.Conf by merging all retrieved maps from the given `locations` in order.
//   - Then applies all the confmap.Converter in the given order.
//
// * Then unmarshalls the confmap.Conf into the service Config.
func NewConfigProvider(set ConfigProviderSettings) (ConfigProvider, error) {
	mr, err := confmap.NewResolver(set.ResolverSettings)
	if err != nil {
		return nil, err
	}

	return &configProvider{
		mapResolver: mr,
		processors:  set.ConfigProcessors,
	}, nil
}

func (cm *configProvider) Get(ctx context.Context, factories Factories) (*Config, error) {
	conf, err := cm.mapResolver.Resolve(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve the configuration: %w", err)
	}

	conf, err = cm.applyConfigProcessors(conf, factories)
	if err != nil {
		return nil, err
	}

	var cfg *configSettings
	if cfg, err = unmarshal(conf, factories); err != nil {
		return nil, fmt.Errorf("cannot unmarshal the configuration: %w", err)
	}

	return &Config{
		Receivers:  cfg.Receivers.Configs(),
		Processors: cfg.Processors.Configs(),
		Exporters:  cfg.Exporters.Configs(),
		Connectors: cfg.Connectors.Configs(),
		Extensions: cfg.Extensions.Configs(),
		Service:    cfg.Service,
	}, nil
}

func (cm *configProvider) applyConfigProcessors(conf *confmap.Conf, factories Factories) (*confmap.Conf, error) {
	if len(cm.processors) == 0 {
		return conf, nil
	}

	// Read processor configs before unmarshaling the
	var processorConfigs map[string]any
	for _, processor := range cm.processors {
		key := processor.ConfigKey()
		subConf, err := conf.Sub(key)
		if err != nil {
			return nil, fmt.Errorf("cannot get config processor configuration %q: %w", key, err)
		}

		processorConf := processor.DefaultConfig()
		err = subConf.Unmarshal(&processorConf)
		if err != nil {
			return nil, fmt.Errorf("cannot unmarshal processor configuration %q: %w", key, err)
		}

		processorConfigs[key] = processorConf
		conf.Delete(key)
	}

	partial, err := unmarshalPartialConfig(conf)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal initial partial config: %w", err)
	}
	for _, processor := range cm.processors {
		processorConf := processorConfigs[processor.ConfigKey()]
		err = processor.Process(partial, factories, processorConf)
		if err != nil {
			return nil, fmt.Errorf("cannot process configuration with processor %q: %w", processor.ConfigKey(), err)
		}
	}

	ret, err := partial.ToConfmap()
	if err != nil {
		return nil, fmt.Errorf("cannot convert resulting partial config: %w", err)
	}
	return ret, nil
}

func (cm *configProvider) Watch() <-chan error {
	return cm.mapResolver.Watch()
}

func (cm *configProvider) Shutdown(ctx context.Context) error {
	return cm.mapResolver.Shutdown(ctx)
}
