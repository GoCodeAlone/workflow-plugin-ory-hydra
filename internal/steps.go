package internal

import (
	"context"
	"fmt"

	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
	hydra "github.com/ory/hydra-client-go/v2"
)

type stepConstructor func(name string, config map[string]any) (sdk.StepInstance, error)

var stepRegistry = map[string]stepConstructor{
	"step.ory_hydra_auth_provider_describe":      newAuthProviderDescribeStep,
	"step.ory_hydra_oauth2_client_create":        newHydraStep(oryHydraOAuth2ClientCreate),
	"step.ory_hydra_oauth2_client_get":           newHydraStep(oryHydraOAuth2ClientGet),
	"step.ory_hydra_oauth2_client_list":          newHydraStep(oryHydraOAuth2ClientList),
	"step.ory_hydra_oauth2_client_update":        newHydraStep(oryHydraOAuth2ClientUpdate),
	"step.ory_hydra_oauth2_client_delete":        newHydraStep(oryHydraOAuth2ClientDelete),
	"step.ory_hydra_json_web_key_set_create":     newHydraStep(oryHydraJSONWebKeySetCreate),
	"step.ory_hydra_json_web_key_set_get":        newHydraStep(oryHydraJSONWebKeySetGet),
	"step.ory_hydra_json_web_key_set_update":     newHydraStep(oryHydraJSONWebKeySetUpdate),
	"step.ory_hydra_json_web_key_set_delete":     newHydraStep(oryHydraJSONWebKeySetDelete),
	"step.ory_hydra_json_web_key_get":            newHydraStep(oryHydraJSONWebKeyGet),
	"step.ory_hydra_json_web_key_update":         newHydraStep(oryHydraJSONWebKeyUpdate),
	"step.ory_hydra_json_web_key_delete":         newHydraStep(oryHydraJSONWebKeyDelete),
	"step.ory_hydra_trusted_jwt_issuer_list":     newHydraStep(oryHydraTrustedJWTIssuerList),
	"step.ory_hydra_trusted_jwt_issuer_get":      newHydraStep(oryHydraTrustedJWTIssuerGet),
	"step.ory_hydra_trusted_jwt_issuer_create":   newHydraStep(oryHydraTrustedJWTIssuerCreate),
	"step.ory_hydra_trusted_jwt_issuer_delete":   newHydraStep(oryHydraTrustedJWTIssuerDelete),
	"step.ory_hydra_oauth2_consent_session_list": newHydraStep(oryHydraOAuth2ConsentSessionList),
}

func allStepTypes() []string {
	return []string{
		"step.ory_hydra_auth_provider_describe",
		"step.ory_hydra_oauth2_client_create",
		"step.ory_hydra_oauth2_client_get",
		"step.ory_hydra_oauth2_client_list",
		"step.ory_hydra_oauth2_client_update",
		"step.ory_hydra_oauth2_client_delete",
		"step.ory_hydra_json_web_key_set_create",
		"step.ory_hydra_json_web_key_set_get",
		"step.ory_hydra_json_web_key_set_update",
		"step.ory_hydra_json_web_key_set_delete",
		"step.ory_hydra_json_web_key_get",
		"step.ory_hydra_json_web_key_update",
		"step.ory_hydra_json_web_key_delete",
		"step.ory_hydra_trusted_jwt_issuer_list",
		"step.ory_hydra_trusted_jwt_issuer_get",
		"step.ory_hydra_trusted_jwt_issuer_create",
		"step.ory_hydra_trusted_jwt_issuer_delete",
		"step.ory_hydra_oauth2_consent_session_list",
	}
}

func createStep(typeName, name string, config map[string]any) (sdk.StepInstance, error) {
	constructor, ok := stepRegistry[typeName]
	if !ok {
		return nil, fmt.Errorf("ory hydra plugin: unknown step type %q", typeName)
	}
	return constructor(name, config)
}

type hydraHandler func(context.Context, *OryHydraClient, map[string]any) (map[string]any, error)

type hydraStep struct {
	name       string
	moduleName string
	handler    hydraHandler
}

func newHydraStep(handler hydraHandler) stepConstructor {
	return func(name string, config map[string]any) (sdk.StepInstance, error) {
		moduleName := stringValue(config, "module")
		if moduleName == "" {
			moduleName = "ory_hydra"
		}
		return &hydraStep{name: name, moduleName: moduleName, handler: handler}, nil
	}
}

func (s *hydraStep) Execute(ctx context.Context, _ map[string]any, _ map[string]map[string]any, current, _, config map[string]any) (*sdk.StepResult, error) {
	client, ok := GetClient(s.moduleName)
	if !ok {
		return &sdk.StepResult{Output: map[string]any{"error": "ory hydra client not found: " + s.moduleName}}, nil
	}
	output, err := s.handler(ctx, client, mergeMaps(config, current))
	if err != nil {
		return &sdk.StepResult{Output: errResult(err)}, nil
	}
	return &sdk.StepResult{Output: output}, nil
}

func oryHydraOAuth2ClientCreate(ctx context.Context, client *OryHydraClient, values map[string]any) (map[string]any, error) {
	body, err := oauth2ClientBody(values)
	if err != nil {
		return nil, err
	}
	created, _, err := client.Admin.OAuth2API.CreateOAuth2Client(ctx).OAuth2Client(body).Execute()
	if err != nil {
		return nil, err
	}
	return encodedObject("client", created)
}

func oryHydraOAuth2ClientGet(ctx context.Context, client *OryHydraClient, values map[string]any) (map[string]any, error) {
	id, err := requiredID(values, "client_id", "clientId", "id")
	if err != nil {
		return nil, err
	}
	found, _, err := client.Admin.OAuth2API.GetOAuth2Client(ctx, id).Execute()
	if err != nil {
		return nil, err
	}
	return encodedObject("client", found)
}

func oryHydraOAuth2ClientList(ctx context.Context, client *OryHydraClient, values map[string]any) (map[string]any, error) {
	req := client.Admin.OAuth2API.ListOAuth2Clients(ctx)
	if name := stringValue(values, "client_name"); name != "" {
		req = req.ClientName(name)
	}
	if owner := stringValue(values, "owner"); owner != "" {
		req = req.Owner(owner)
	}
	if pageSize := intValue(values, "page_size", 0); pageSize > 0 {
		req = req.PageSize(int64(pageSize))
	}
	if pageToken := stringValue(values, "page_token"); pageToken != "" {
		req = req.PageToken(pageToken)
	}
	clients, _, err := req.Execute()
	if err != nil {
		return nil, err
	}
	return encodedValueWithKey("clients", clients)
}

func oryHydraOAuth2ClientUpdate(ctx context.Context, client *OryHydraClient, values map[string]any) (map[string]any, error) {
	id, err := requiredID(values, "client_id", "clientId", "id")
	if err != nil {
		return nil, err
	}
	body, err := oauth2ClientBody(values)
	if err != nil {
		return nil, err
	}
	updated, _, err := client.Admin.OAuth2API.SetOAuth2Client(ctx, id).OAuth2Client(body).Execute()
	if err != nil {
		return nil, err
	}
	return encodedObject("client", updated)
}

func oryHydraOAuth2ClientDelete(ctx context.Context, client *OryHydraClient, values map[string]any) (map[string]any, error) {
	id, err := requiredID(values, "client_id", "clientId", "id")
	if err != nil {
		return nil, err
	}
	if _, err := client.Admin.OAuth2API.DeleteOAuth2Client(ctx, id).Execute(); err != nil {
		return nil, err
	}
	return map[string]any{"deleted": true, "client_id": id}, nil
}

func oryHydraJSONWebKeySetCreate(ctx context.Context, client *OryHydraClient, values map[string]any) (map[string]any, error) {
	set, err := requiredID(values, "set", "set_id", "setId")
	if err != nil {
		return nil, err
	}
	body, err := createJSONWebKeySetBody(values)
	if err != nil {
		return nil, err
	}
	jwkSet, _, err := client.Admin.JwkAPI.CreateJsonWebKeySet(ctx, set).CreateJsonWebKeySet(body).Execute()
	if err != nil {
		return nil, err
	}
	return encodedObject("jwk_set", jwkSet)
}

func oryHydraJSONWebKeySetGet(ctx context.Context, client *OryHydraClient, values map[string]any) (map[string]any, error) {
	set, err := requiredID(values, "set", "set_id", "setId")
	if err != nil {
		return nil, err
	}
	jwkSet, _, err := client.Admin.JwkAPI.GetJsonWebKeySet(ctx, set).Execute()
	if err != nil {
		return nil, err
	}
	return encodedObject("jwk_set", jwkSet)
}

func oryHydraJSONWebKeySetUpdate(ctx context.Context, client *OryHydraClient, values map[string]any) (map[string]any, error) {
	set, err := requiredID(values, "set", "set_id", "setId")
	if err != nil {
		return nil, err
	}
	body, err := jsonWebKeySetBody(values)
	if err != nil {
		return nil, err
	}
	jwkSet, _, err := client.Admin.JwkAPI.SetJsonWebKeySet(ctx, set).JsonWebKeySet(body).Execute()
	if err != nil {
		return nil, err
	}
	return encodedObject("jwk_set", jwkSet)
}

func oryHydraJSONWebKeySetDelete(ctx context.Context, client *OryHydraClient, values map[string]any) (map[string]any, error) {
	set, err := requiredID(values, "set", "set_id", "setId")
	if err != nil {
		return nil, err
	}
	if _, err := client.Admin.JwkAPI.DeleteJsonWebKeySet(ctx, set).Execute(); err != nil {
		return nil, err
	}
	return map[string]any{"deleted": true, "set": set}, nil
}

func oryHydraJSONWebKeyGet(ctx context.Context, client *OryHydraClient, values map[string]any) (map[string]any, error) {
	set, kid, err := requiredSetAndKid(values)
	if err != nil {
		return nil, err
	}
	jwkSet, _, err := client.Admin.JwkAPI.GetJsonWebKey(ctx, set, kid).Execute()
	if err != nil {
		return nil, err
	}
	return encodedObject("jwk_set", jwkSet)
}

func oryHydraJSONWebKeyUpdate(ctx context.Context, client *OryHydraClient, values map[string]any) (map[string]any, error) {
	set, kid, err := requiredSetAndKid(values)
	if err != nil {
		return nil, err
	}
	body, err := jsonWebKeyBody(values)
	if err != nil {
		return nil, err
	}
	jwk, _, err := client.Admin.JwkAPI.SetJsonWebKey(ctx, set, kid).JsonWebKey(body).Execute()
	if err != nil {
		return nil, err
	}
	return encodedObject("jwk", jwk)
}

func oryHydraJSONWebKeyDelete(ctx context.Context, client *OryHydraClient, values map[string]any) (map[string]any, error) {
	set, kid, err := requiredSetAndKid(values)
	if err != nil {
		return nil, err
	}
	if _, err := client.Admin.JwkAPI.DeleteJsonWebKey(ctx, set, kid).Execute(); err != nil {
		return nil, err
	}
	return map[string]any{"deleted": true, "set": set, "kid": kid}, nil
}

func oryHydraTrustedJWTIssuerList(ctx context.Context, client *OryHydraClient, _ map[string]any) (map[string]any, error) {
	issuers, _, err := client.Admin.OAuth2API.ListTrustedOAuth2JwtGrantIssuers(ctx).Execute()
	if err != nil {
		return nil, err
	}
	return encodedValueWithKey("trusted_jwt_issuers", issuers)
}

func oryHydraTrustedJWTIssuerGet(ctx context.Context, client *OryHydraClient, values map[string]any) (map[string]any, error) {
	id, err := requiredID(values, "issuer_id", "issuerId", "id")
	if err != nil {
		return nil, err
	}
	issuer, _, err := client.Admin.OAuth2API.GetTrustedOAuth2JwtGrantIssuer(ctx, id).Execute()
	if err != nil {
		return nil, err
	}
	return encodedObject("trusted_jwt_issuer", issuer)
}

func oryHydraTrustedJWTIssuerCreate(ctx context.Context, client *OryHydraClient, values map[string]any) (map[string]any, error) {
	var body hydra.TrustOAuth2JwtGrantIssuer
	source := values
	if payload := mapValue(values, "trusted_jwt_issuer"); payload != nil {
		source = payload
	}
	if err := decodeMap(source, &body); err != nil {
		return nil, err
	}
	issuer, _, err := client.Admin.OAuth2API.TrustOAuth2JwtGrantIssuer(ctx).TrustOAuth2JwtGrantIssuer(body).Execute()
	if err != nil {
		return nil, err
	}
	return encodedObject("trusted_jwt_issuer", issuer)
}

func oryHydraTrustedJWTIssuerDelete(ctx context.Context, client *OryHydraClient, values map[string]any) (map[string]any, error) {
	id, err := requiredID(values, "issuer_id", "issuerId", "id")
	if err != nil {
		return nil, err
	}
	if _, err := client.Admin.OAuth2API.DeleteTrustedOAuth2JwtGrantIssuer(ctx, id).Execute(); err != nil {
		return nil, err
	}
	return map[string]any{"deleted": true, "issuer_id": id}, nil
}

func oryHydraOAuth2ConsentSessionList(ctx context.Context, client *OryHydraClient, values map[string]any) (map[string]any, error) {
	subject, err := requiredID(values, "subject")
	if err != nil {
		return nil, err
	}
	req := client.Admin.OAuth2API.ListOAuth2ConsentSessions(ctx).Subject(subject)
	if loginSessionID := stringValue(values, "login_session_id"); loginSessionID != "" {
		req = req.LoginSessionId(loginSessionID)
	}
	if pageSize := intValue(values, "page_size", 0); pageSize > 0 {
		req = req.PageSize(int64(pageSize))
	}
	if pageToken := stringValue(values, "page_token"); pageToken != "" {
		req = req.PageToken(pageToken)
	}
	sessions, _, err := req.Execute()
	if err != nil {
		return nil, err
	}
	return encodedValueWithKey("consent_sessions", sessions)
}

func oauth2ClientBody(values map[string]any) (hydra.OAuth2Client, error) {
	source := values
	if payload := mapValue(values, "client"); payload != nil {
		source = payload
	}
	var body hydra.OAuth2Client
	if err := decodeMap(source, &body); err != nil {
		return hydra.OAuth2Client{}, err
	}
	if body.ClientName == nil && body.ClientId == nil {
		return hydra.OAuth2Client{}, fmt.Errorf("client_name or client_id is required")
	}
	return body, nil
}

func createJSONWebKeySetBody(values map[string]any) (hydra.CreateJsonWebKeySet, error) {
	source := values
	if payload := mapValue(values, "jwk_set_create"); payload != nil {
		source = payload
	}
	var body hydra.CreateJsonWebKeySet
	if err := decodeMap(source, &body); err != nil {
		return hydra.CreateJsonWebKeySet{}, err
	}
	if body.Alg == "" {
		return hydra.CreateJsonWebKeySet{}, fmt.Errorf("alg is required")
	}
	if body.Kid == "" {
		return hydra.CreateJsonWebKeySet{}, fmt.Errorf("kid is required")
	}
	if body.Use == "" {
		return hydra.CreateJsonWebKeySet{}, fmt.Errorf("use is required")
	}
	return body, nil
}

func jsonWebKeySetBody(values map[string]any) (hydra.JsonWebKeySet, error) {
	source := values
	if payload := mapValue(values, "jwk_set"); payload != nil {
		source = payload
	}
	var body hydra.JsonWebKeySet
	if err := decodeMap(source, &body); err != nil {
		return hydra.JsonWebKeySet{}, err
	}
	if len(body.Keys) == 0 {
		return hydra.JsonWebKeySet{}, fmt.Errorf("keys is required")
	}
	return body, nil
}

func jsonWebKeyBody(values map[string]any) (hydra.JsonWebKey, error) {
	source := values
	if payload := mapValue(values, "jwk"); payload != nil {
		source = payload
	}
	var body hydra.JsonWebKey
	if err := decodeMap(source, &body); err != nil {
		return hydra.JsonWebKey{}, err
	}
	return body, nil
}

func requiredID(values map[string]any, keys ...string) (string, error) {
	id := firstNonEmpty(values, keys...)
	if id == "" {
		return "", fmt.Errorf("%s is required", keys[0])
	}
	return id, nil
}

func requiredSetAndKid(values map[string]any) (string, string, error) {
	set, err := requiredID(values, "set", "set_id", "setId")
	if err != nil {
		return "", "", err
	}
	kid, err := requiredID(values, "kid", "key_id", "keyId")
	if err != nil {
		return "", "", err
	}
	return set, kid, nil
}

func firstNonEmpty(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := stringValue(values, key); value != "" {
			return value
		}
	}
	return ""
}

func encodedObject(key string, value any) (map[string]any, error) {
	encoded, err := encodeValue(value)
	if err != nil {
		return nil, err
	}
	return map[string]any{key: encoded}, nil
}

func encodedValueWithKey(key string, value any) (map[string]any, error) {
	encoded, err := encodeAny(value)
	if err != nil {
		return nil, err
	}
	return map[string]any{key: encoded}, nil
}
