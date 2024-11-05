// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import "go.opentelemetry.io/collector/confmap"

type ConfigProcessor interface {
	ConfigKey() string
	DefaultConfig() any
	Process(conf *PartialConfig, factories Factories, processorConf any) error
}

type PartialConfig Config

func (c *PartialConfig) ToConfmap() (*confmap.Conf, error) {
	ret := confmap.New()
	return ret, ret.Marshal(c)
}

// unmarshalPartialConfig decodes configuration without performing any validation.
func unmarshalPartialConfig(conf *confmap.Conf) (*PartialConfig, error) {
	var c PartialConfig
	return &c, conf.Unmarshal(&c)
}
