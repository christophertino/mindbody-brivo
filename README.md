# MINDBODY / Brivo OnAir Membership Sync

Sync membership data between MINDBODY and Brivo OnAir. This application makes the following assumptions:

+ Master client list is stored in MINDBODY and mirrored to Brivo
+ MINDBODY users have been assigned a wristband with a generated barcode ID
+ Brivo credentials are cleared when the user is deactivated in MINDBODY

## Generating API Keys

+ [Brivo OnAir API](https://developer.brivo.com/)
+ [MINDBODY API](https://developers.mindbodyonline.com/)

### Create MINDBODY Webhook Subscriptions

This app requires active webhook subscriptions for:

+ client.created
+ client.updated
+ client.deactivated

See [Webhook Subscriptions](https://developers.mindbodyonline.com/WebhooksDocumentation#subscriptions) documentation.

## Running the App

```sh
# If running inside $GOPATH
$ export GO111MODULE=on
```

```sh
# Install dependencies with Go Modules
$ go mod init
$ go build cmd/mindbody-brivo/main.go
```

```sh
# Generate configuration file
$ cp conf/conf.example.json conf/conf.json
```

### Provision Brivo OnAir 

```sh
# On first run, copy all MINDBODY users to Brivo
$ go run cmd/sync/main.go
```

### Start API Server

```sh
# Run the application and listen for webhook events
$ go run cmd/webhook/main.go
```

## License
[MPL-2.0](https://www.mozilla.org/en-US/MPL/2.0/)

See [LICENSE](LICENSE)