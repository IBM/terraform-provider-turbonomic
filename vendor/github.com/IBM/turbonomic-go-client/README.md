# Turbonomic-Go-Client

Use this simple GoLang Library to access the Turbonomic API. It currently supports authentication by using a username and password.

## Requirements

- [Go](https://golang.org/doc/install) >= 1.23.7

## Authenticating to the Turbonomic API
You can authenticate to the Turbonomic API by using either a username and password or oAuth 2.0.

### Authenticating with a username and password
This library creates a client object by passing in a `ClientParameters` struct with the following parameters:

- Hostname
- Username
- Password
- ApiInfo

Create a client with these parameters, similar to the following example:

```
newClientOpts := ClientParameters{
    Hostname:   "TurboHostname",
    Username:   "TurboUsername",
    Password:   "TurboPassword",
    ApiInfo: ApiInfo{
        ApiOrigin: "your-app-name", // e.g., "terraform-provider"
        Version:   "version",         // version of your client or integration e.g : 1.0.0
    },
}

turboClient, err := NewClient(&newClientOpts)
if err != nil {
    panic(err)
}
```

You can then use this client to call other methods to interact with the Turbonomic API.

### Authenticating with oAuth 2.0

In order to authenticate to Turbonomic's API using oAuth 2,0, you first need to create an oAuth client.  Follow [Creating and authenticating an OAuth 2.0 client](https://www.ibm.com/docs/en/tarm/8.15.0?topic=cookbook-authenticating-oauth-20-clients-api#cookbook_administration_oauth_authentication__title__4)
to create the client.  The output from this will be the following parameters:
- clientId
- clientSecret
- role

Once you have the prerequisite parameters, you will want to create a `OAuthCreds` struct similar to the following example:

```
oauthCreds := OAuthCreds{
    ClientId:     clientId,
    ClientSecret: clientSecret,
    Role:         role,
}
```

**Note:** Valid roles are ADMINISTRATOR, SITE_ADMIN, AUTOMATOR, DEPLOYER, ADVISOR, OBSERVER, OPERATIONAL_OBSERVER, SHARED_ADVISOR, SHARED_OBSERVER, REPORT_EDITOR.

You then pass the `OAuthCreds` struct with the Hostname of your Turbonomic instance to a `ClientParameters` struct to create a Turbonomic client:

```
    newClientOpts := ClientParameters{
        Hostname:   TurboHost,
        OAuthCreds: oauthCreds,
        ApiInfo: ApiInfo{
            ApiOrigin: "your-app-name",
            Version:   "vesrion",
        },
    }

    if err != nil {
        panic(err)
    }
```

You can then use this client to call other methods to interact with the Turbonomic API.

### Using a self-signed certificate
If your server has a self-signed certificate, you can skip SSL validation by also passing in the `Skipverify` parameter in the `ClientParameters` struct:

```
newClientOpts := ClientParameters{
    Hostname: "TurboHostname",
    Username: "TurboUsername",
    Password: "TurboPassword",
    Skipverify: true
    }
```

## Searching for an entity by name

To search for an entity by name, pass a `SearchRequest` struct to the `SearchEntityByName` method:

```
    searchReq := SearchRequest{
        Name:            "VM_Test_1",
        EntityType:      "VirtualMachine",
        EnvironmentType: "ONPREM",
        CaseSensitive:   true,
    }

    entityName, err := c.SearchEntityByName(searchReq)
```

## Retrieving an entity by UUID

To retrieve entity data based on its UUID, pass a `EntityRequest` struct to the `GetEntity` method:

```
    entityReq := EntityRequest{Uuid: "123456789"}

    entityName, err := c.GetEntity(entityReq)
```

## Retrieving actions based on request parameters and entity UUID

To retrieve action data, pass an `ActionsRequest` struct to the `GetActions` method:

```
    actionReq := ActionsRequest{
        Uuid: "123456789",
        ActionState: []string{"READY", "ACCEPTED", "QUEUED", "IN_PROGRESS"},
        ActionType: []string{"RESIZE"},
    }

    actions, err := GetActionsByUUID(actionReq)
```

## Logging

Additional logging can be enabled via the `T8C_LOG` environment variable.  Valid values are:

- `INFO`
- `DEBUG`
- `WARN`
- `ERROR`

Note, the values are case insensitive.

### Example

```
export T8C_LOG=debug
```
