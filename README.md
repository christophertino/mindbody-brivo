# FIAO Brooklyn Membership Sync

Sync membership data between MindBody and Brivo OnAir. This application makes the following assumptions:

+ Master client list is stored in MindBody and mirrored to Brivo
+ Mindbody users have been assigned a wristband with a generated barcode ID
+ Brivo credentials are cleard when the user is deactivated in MindBody

Built for the Federation of Italian-American Organizations of Brooklyn. 

## Generating API Keys

+ [Brivo OnAir API](https://developer.brivo.com/)
+ [MindBody API](https://developers.mindbodyonline.com/)

### Create MindBody Webhook Subscriptions

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
# Install dependencies with Go modules
$ go mod init
$ go build cmd/fiao/main.go
```

```sh
# Generate configuration file
$ cp conf/conf.example.json conf/conf.json
```

```sh
# On first run, copy all MindBody users to Brivo
$ go run cmd/fiao/main.go provision
```

```sh
# Run the application and listen for webhook events
$ go run cmd/fiao/main.go
```

## License
[MPL-2.0](https://www.mozilla.org/en-US/MPL/2.0/) Copyright 2019 FIAO Brooklyn. All rights reserved.

See [LICENSE](LICENSE)