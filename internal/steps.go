package internal

import (
	"context"

	"github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

type StreamStartStep struct {
	config map[string]any
}

func (s *StreamStartStep) Execute(
	ctx context.Context,
	triggerData map[string]any,
	stepOutputs map[string]map[string]any,
	current map[string]any,
	metadata map[string]any,
	config map[string]any,
) (*sdk.StepResult, error) {
	_ = ctx
	_ = triggerData
	_ = stepOutputs
	_ = current
	_ = metadata
	_ = config
	return &sdk.StepResult{Output: map[string]any{
		"status": "pending-provider-contract",
	}}, nil
}

type StreamRestreamStep struct {
	config map[string]any
}

func (s *StreamRestreamStep) Execute(
	ctx context.Context,
	triggerData map[string]any,
	stepOutputs map[string]map[string]any,
	current map[string]any,
	metadata map[string]any,
	config map[string]any,
) (*sdk.StepResult, error) {
	_ = ctx
	_ = triggerData
	_ = stepOutputs
	_ = current
	_ = metadata
	_ = config
	return &sdk.StepResult{Output: map[string]any{
		"status": "pending-provider-contract",
	}}, nil
}
