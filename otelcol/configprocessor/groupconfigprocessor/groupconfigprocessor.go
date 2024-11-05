// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package groupconfigprocessor // import "go.opentelemetry.io/collector/otelcol/configprocessor/configprocessorgroup"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/otelcol"
)

func New() otelcol.ConfigProcessor {
	return &groupConfigProcessor{}
}

type groupConfigProcessor struct{}

type groupConfigProcessorConfig struct{}

func (p *groupConfigProcessor) ConfigKey() string {
	return "group"
}

func (p *groupConfigProcessor) DefaultConfig() any {
	return &groupConfigProcessorConfig{}
}

func (p *groupConfigProcessor) Process(conf *otelcol.PartialConfig, _ otelcol.Factories, _ any) error {
	groups := make(map[component.ID][]component.ID)
	for id, receiver := range conf.Receivers {
		if id.Type() != "group" {
			continue
		}

		subcomponents, ok := receiver.(map[string]any)
		for id, subcomponent := range subcomponents {
		}

		delete(conf.Receivers, id)
	}
}
