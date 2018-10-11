# cf-eureka
Registers CloudFoundry hosted services with local eureka instance for convenient debugging

# cf-app-routes
This tool uses the `cf` CloudFoundry CLI tool to query for apps and their default routes for a given space. To install the `cf` CLI tool, go here: https://docs.cloudfoundry.org/cf-cli/install-go-cli.html#pkg

To build it,

```
$ cd cf-app-routes
$ go build
```

To run it,

```
$ ./cf-app-routes nu dev > dev.json
```

The first argument is the name of the CF "org" to query routes for.

The second argument is the name of the CF "space" to query routes for.

Both the org name and space name are required to avoid naming conflicts among spaces in multiple orgs.

The tool assumes you are already logged in with `cf login` and that the `cf curl` function works and can make authenticated API requests.

The tool outputs a JSON dictionary of the form `{ "appName": "host", ... }` to stdout, associating each register application with its default public host name from the routing table (e.g. `app-name-hairy-bear.your.domain`).

It is wise to redirect stdout to a file to capture the resulting JSON output so it can be passed to `eureka-register` tool, described below.

# eureka-register
This tool takes as input the JSON dictionary of the `cf-app-routes` tool described above, and registers each app with a running instance of `eureka` (compatible with up to 1.4.x) pointing to the public host name.

To build it,
```
$ cd eureka-register
$ go build
```

To run it,
```
$ ./eureka-register dev.json
```

This tool assumes two environment variables exist, namely EUREKA_HOST and EUREKA_PORT, which refer to the running instance of eureka's hostname and port, respectively. The defaults are assumed to be `localhost` and `8080`.

To override the environment variables while running the tool, do this:
```
$ EUREKA_HOST=localhost EUREKA_PORT=8080 ./eureka-register dev.json
```

The output of the tool should show success/failure messages per each application registration operation.
