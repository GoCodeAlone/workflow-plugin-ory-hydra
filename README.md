# workflow-plugin-ory-hydra

Ory Hydra OAuth2/OpenID Connect provider plugin for Workflow. It uses the
official `github.com/ory/hydra-client-go/v2` SDK.

## Capabilities

- `ory.hydra` module using the Hydra Admin API base URL
- Auth provider descriptor step for admin catalog integration
- OAuth2/OIDC client create/read/list/update/delete steps
- JSON Web Key set and key create/read/update/delete steps
- Trusted JWT bearer grant issuer management steps
- OAuth2 consent session listing for access audits

The descriptor advertises only capabilities backed by the plugin's concrete
management steps.

Hydra is not an identity-management or MFA provider. Use `workflow-plugin-ory-kratos`
for Ory identity/self-service flows and an SSO/enterprise plugin for SAML,
SCIM, or directory synchronization.

## Install

```sh
wfctl plugin install workflow-plugin-ory-hydra
```

## License

MIT
