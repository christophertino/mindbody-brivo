# MINDBODY / Brivo OnAir Membership Sync

Sync membership data between MINDBODY and Brivo OnAir. This application makes the following assumptions:

+ The master client data is stored in MINDBODY and mirrored to Brivo
+ MINDBODY users have been assigned a wristband with a generated hexadecimal ID
    + Only users with a valid hex ID will be mirrored to Brivo
+ When a user is deactivated in MINDBODY, their account is put into suspended state in Brivo

#### Setting up Brivo OnAir

+ Create a "Members" user group and add the GroupID to [.env](.env)
+ Create a Custom Field called `Barcode ID` of type `Text` and add the FieldID to [.env](.env)
+ Create a Custom Field called `User Type` of type `Text` and add the FieldID to [.env](.env)

## Generating API Keys

+ [Brivo OnAir API](https://developer.brivo.com/)
+ [MINDBODY API](https://developers.mindbodyonline.com/)

### Create MINDBODY Webhook Subscriptions

This app requires active webhook subscriptions for:

+ client.created
+ client.updated
+ client.deactivated

See [Webhook Subscriptions](https://developers.mindbodyonline.com/WebhooksDocumentation#subscriptions) documentation.

For validation, we use the `X-Mindbody-Signature` header and the `messageSignatureKey` returned from the `POST` Subscription webhook endpoint. [Read more](https://developers.mindbodyonline.com/WebhooksDocumentation?shell#x-mindbody-signature-header) 

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
# Generate a local configuration file
$ cp .env-example .env
```

### Provision Brivo OnAir 

```sh
# On first run, migrate all MINDBODY users to Brivo
$ go run cmd/migrate/main.go
```

### Start API Server

```sh
# Run the application and listen for webhook events
$ go run cmd/server/main.go
```

## Heroku Integration

+ Install the [Heroku CLI](https://devcenter.heroku.com/articles/heroku-cli)
+ Create config vars from [.env](.env) on Heroku [link](https://devcenter.heroku.com/articles/config-vars#managing-config-vars)

#### Developing Locally

```sh
# Compile the server application
$ go build -o bin/server -v cmd/server/main.go
```

```sh
# Run server application locally
$ heroku local web
```

#### Deploying to Heroku
```sh
$ git push heroku master
```

### Clear Brivo OnAir Development Environment

```sh
# This will remove all users and credentials from Brivo
$ go run cmd/clean/main.go
```

## License
[MPL-2.0](https://www.mozilla.org/en-US/MPL/2.0/)

See [LICENSE](LICENSE)