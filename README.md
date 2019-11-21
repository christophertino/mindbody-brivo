# MINDBODY / Brivo OnAir Membership Management

Coordinate membership data and access control between MINDBODY and Brivo OnAir. This application makes the following assumptions:

+ The master client data is stored in MINDBODY and mirrored to Brivo
+ MINDBODY users have been assigned a wristband with an ID format of `FACILITY_CODE-MEMBER_ID`
    + Brivo facility codes are used to group members into access groups
    + Only users with a valid ID format will be mirrored to Brivo
+ When a user is deactivated in MINDBODY, their account is put into suspended state in Brivo

#### Facility Access Control

The application supports access control in two scenarios:

1. On-Site Membership Check-in - A user scans his/her wristband at the facility counter. This triggers a MINDBODY Webhook to update Brivo with new membership data, if necessary.
2. External Access Points - A user scans his/her wristband to enter the facility (locked door, parking garage, etc). This triggers a Brivo Event which updates MINDBODY of the user check-in

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

### Create Brivo Event Subscriptions

Create Event Subscriptions for each of the Brivo Access Points you want the application to monitor. 

```json
{
  "name" : "Event Name",
  "url" : "https://your-application-url",
  "errorEmail": "you@error-email.com",
  "criteria": [{
            "criteriaType": "ACCESS-POINT",
            "criteriaOperationType": "EQ",
            "criteriaValue": 12345
        }, {
            "criteriaType": "ACCESS-POINT",
            "criteriaOperationType": "EQ",
            "criteriaValue": 67890
        }
    ]
}
```
See [Event Subscription](https://apidocs.brivo.com/#api-Event_Subscription) documentation.

## Running the App

```sh
# If running inside $GOPATH
$ export GO111MODULE=on
```

```sh
# Install dependencies with Go Modules
$ go mod init
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

See [LICENSE](LICENSE)