---
title: Usage
expires_at: never
tags: [routing-release,routing-api]
---

# Usage

## Server Configuration

### jwt token

To run the routing-api server, a configuration file with the public uaa jwt token must be provided.
This configuration file can then be passed in with the flag `-config [path_to_config]`.
An example of the configuration file can be found under `example_config/example.yml` for bosh-lite.

To generate your own config file, you must provide a `uaa_verification_key` in
pem format, such as the following:

```yaml
uaa_verification_key: |
  -----BEGIN PUBLIC KEY-----
  SOME_KEY
  -----END PUBLIC KEY-----
```

This can be found in your Cloud Foundry manifest under `uaa.jwt.verification_key`

### Oauth Clients

The Routing API uses OAuth tokens to authenticate clients. To obtain a token
from UAA that grants the API client permission to register routes, an OAuth
client must first be created for the API client in UAA. An API client can then
authenticate with UAA using the registered OAuth client credentials, request a
token, then provide this token with requests to the Routing API.

Registering OAuth clients can be done using the cf-release BOSH deployment
manifest, or manually using the `uaac` CLI for UAA.

- For API clients that wish to register/unregister routes with the Routing API,
  the OAuth client in UAA must be configured with the `routing.routes.write`
  authority.
- For API clients that wish to list routes with the Routing API, the OAuth
  client in UAA must be configured with the `routing.routes.read` authority.
- For API clients that wish to list router groups with the Routing API, the
  OAuth client in UAA must be configured with the `routing.router_groups.read`
  authority.

For instructions on fetching a token, please refer to the [API
documentation](./02-api-docs.md).

### Configure OAuth clients in the cf-release BOSH Manifest

```yaml
uaa:
  clients:
    routing_api_client:
      authorities: routing.routes.write,routing.routes.read,routing.router_groups.read
      authorized_grant_type: client_credentials
      secret: route_secret
```

### Configure OAuth clients manually using `uaac` CLI for UAA

1. Install the `uaac` CLI

   ```bash
   gem install cf-uaac
   ```

2. Get the admin client token

   ```bash
   uaac target uaa.bosh-lite.com
   uaac token client get admin # You will need to provide the client_secret, found in your CF manifest.
   ```

3. Create the OAuth client.

   ```bash
   uaac client add routing_api_client \
     --authorities "routing.routes.write,routing.routes.read,routing.router_groups.read" \
     --authorized_grant_type "client_credentials"
   ```

### Starting the Server

To run the API server you need to provide RDB configuration for the Postgres or
MySQL, a configuration file containing the public UAA jwt key, plus some
optional flags.

```bash
routing-api \
   -ip 127.0.0.1 \
   -systemDomain 127.0.0.1.xip.io \
   -config example_config/example.yml \
   -port 3000 \
   -maxTTL 60
```


### Profiling the Server

The Routing API runs the
[cf_debug_server](https://github.com/cloudfoundry/debugserver), which is a
wrapper around the go pprof tool. In order to generate this profile, do the
following:

```bash
# Establish a SSH tunnel to your server (not necessary if you can connect directly)
ssh -L localhost:8080:[INTERNAL_SERVER_IP]:17002 vcap@[BOSH_DIRECTOR]
# Run the profile tool.
go tool pprof http://localhost:8080/debug/pprof/profile
```

> Note: Debug server should run on loopback interface i.e., 0.0.0.0 for the SSH
> tunnel to work.

## Using the API

The Routing API uses OAuth tokens to authenticate clients. To obtain a token
from UAA an OAuth client must first be created for the API client in UAA. For
instructions on registering OAuth clients, see [Server
Configuration](#oauth-clients).
