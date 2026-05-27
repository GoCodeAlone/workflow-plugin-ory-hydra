package internal

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/GoCodeAlone/workflow-plugin-ory-hydra/internal/contracts"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func typedModuleConfig(cfg *contracts.ProviderConfig) map[string]any {
	if cfg == nil {
		return map[string]any{}
	}
	values := map[string]any{}
	if cfg.GetAdminUrl() != "" {
		values["adminUrl"] = cfg.GetAdminUrl()
	}
	if cfg.GetApiKey() != "" {
		values["apiKey"] = cfg.GetApiKey()
	}
	return values
}

func typedStepHandler(typeName string) sdk.TypedStepHandler[*contracts.OryHydraStepConfig, *contracts.OryHydraStepInput, *contracts.OryHydraStepOutput] {
	return func(ctx context.Context, req sdk.TypedStepRequest[*contracts.OryHydraStepConfig, *contracts.OryHydraStepInput]) (*sdk.TypedStepResult[*contracts.OryHydraStepOutput], error) {
		config, err := structToMap(req.Config.GetValues())
		if err != nil {
			return nil, err
		}
		if req.Config.GetModule() != "" {
			config["module"] = req.Config.GetModule()
		}
		input, err := structToMap(req.Input.GetValues())
		if err != nil {
			return nil, err
		}
		step, err := createStep(typeName, "typed", config)
		if err != nil {
			return nil, err
		}
		result, err := step.Execute(ctx, req.TriggerData, req.StepOutputs, mergeMaps(req.Current, input), req.Metadata, config)
		if err != nil {
			return nil, err
		}
		output, err := mapToStruct(result.Output)
		if err != nil {
			return nil, err
		}
		return &sdk.TypedStepResult[*contracts.OryHydraStepOutput]{
			Output:       &contracts.OryHydraStepOutput{Values: output, StopPipeline: result.StopPipeline},
			StopPipeline: result.StopPipeline,
		}, nil
	}
}

func typedAuthProviderDescribe(ctx context.Context, req sdk.TypedStepRequest[*contracts.AuthProviderDescribeConfig, *contracts.AuthProviderDescribeInput]) (*sdk.TypedStepResult[*contracts.AuthProviderDescribeOutput], error) {
	config := map[string]any{}
	if req.Config.GetProviderId() != "" {
		config["provider_id"] = req.Config.GetProviderId()
	}
	if req.Config.GetAdminUrl() != "" {
		config["admin_url"] = req.Config.GetAdminUrl()
	}
	current := map[string]any{}
	if req.Input.GetProviderId() != "" {
		current["provider_id"] = req.Input.GetProviderId()
	}
	if req.Input.GetAdminUrl() != "" {
		current["admin_url"] = req.Input.GetAdminUrl()
	}
	step, err := newAuthProviderDescribeStep("typed", config)
	if err != nil {
		return nil, err
	}
	result, err := step.Execute(ctx, req.TriggerData, req.StepOutputs, current, req.Metadata, nil)
	if err != nil {
		return nil, err
	}
	output, err := mapToProtoMessage(result.Output, &contracts.AuthProviderDescribeOutput{})
	if err != nil {
		return nil, err
	}
	return &sdk.TypedStepResult[*contracts.AuthProviderDescribeOutput]{Output: output}, nil
}

func structToMap(value *structpb.Struct) (map[string]any, error) {
	if value == nil {
		return map[string]any{}, nil
	}
	data, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal typed struct: %w", err)
	}
	values := map[string]any{}
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("decode typed struct: %w", err)
	}
	return values, nil
}

func mapToStruct(values map[string]any) (*structpb.Struct, error) {
	if values == nil {
		values = map[string]any{}
	}
	data, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("marshal output map: %w", err)
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, fmt.Errorf("decode output map: %w", err)
	}
	out, err := structpb.NewStruct(decoded)
	if err != nil {
		return nil, fmt.Errorf("encode typed output struct: %w", err)
	}
	return out, nil
}

func mapToProtoMessage[O proto.Message](values map[string]any, target O) (O, error) {
	typed := proto.Clone(target).(O)
	data, err := json.Marshal(values)
	if err != nil {
		return typed, fmt.Errorf("marshal output map: %w", err)
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(data, typed); err != nil {
		return typed, fmt.Errorf("decode typed protobuf output: %w", err)
	}
	return typed, nil
}
