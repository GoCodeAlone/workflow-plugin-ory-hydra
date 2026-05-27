.PHONY: build test cross-build clean generate-contracts check-contracts validate-contracts

BINARY_NAME = workflow-plugin-ory-hydra
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS = -ldflags "-X github.com/GoCodeAlone/workflow-plugin-ory-hydra/internal.Version=$(VERSION)"
PLATFORMS = linux/amd64 linux/arm64 darwin/amd64 darwin/arm64

build:
	GOWORK=off CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/$(BINARY_NAME)

test:
	GOWORK=off go test ./...

cross-build:
	@mkdir -p bin
	@for platform in $(PLATFORMS); do \
		os=$${platform%%/*}; \
		arch=$${platform##*/}; \
		output=bin/$(BINARY_NAME)-$${os}-$${arch}; \
		echo "Building $${output}..."; \
		GOWORK=off CGO_ENABLED=0 GOOS=$${os} GOARCH=$${arch} \
			go build $(LDFLAGS) -o $${output} ./cmd/$(BINARY_NAME); \
	done

generate-contracts:
	jq '{version:"1",contracts:(([.capabilities.moduleTypes[]|{kind:"module",type:.,mode:"strict",config:"workflow.plugins.ory_hydra.v1.ProviderConfig"}]+[.capabilities.stepTypes[]|if . == "step.ory_hydra_auth_provider_describe" then {kind:"step",type:.,mode:"strict",config:"workflow.plugins.ory_hydra.v1.AuthProviderDescribeConfig",input:"workflow.plugins.ory_hydra.v1.AuthProviderDescribeInput",output:"workflow.plugins.ory_hydra.v1.AuthProviderDescribeOutput"} else {kind:"step",type:.,mode:"strict",config:"workflow.plugins.ory_hydra.v1.OryHydraStepConfig",input:"workflow.plugins.ory_hydra.v1.OryHydraStepInput",output:"workflow.plugins.ory_hydra.v1.OryHydraStepOutput"} end]))}' plugin.json > plugin.contracts.json

check-contracts: generate-contracts
	git diff --exit-code -- plugin.contracts.json

validate-contracts:
	wfctl plugin validate-contract .

clean:
	rm -rf bin dist
