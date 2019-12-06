# MINDBODY / Brivo OnAir Membership Management

Coordinate membership data and access control between MINDBODY and Brivo OnAir. This application makes the following assumptions:

+ The master client data is stored in MINDBODY and mirrored to Brivo
+ MINDBODY users have been assigned a wristband with an ID format of `FACILITY_CODE-MEMBER_ID`
    + Brivo facility codes are used to organize members into access groups
    + Only users with a valid ID format will be mirrored to Brivo
+ When a user is deactivated in MINDBODY, their account is put into suspended state in Brivo

#### Facility Access Control

The application supports access control in two scenarios:

1. MINDBODY On-Site Check-In: A user scans his/her wristband at the facility counter using a MINDBODY reader. This triggers a MINDBODY webhook which updates Brivo with new membership data, if necessary.
2. Brivo External Access Points: A user scans his/her wristband to enter the facility via a Brivo access point (locked door, parking garage, etc). This triggers a Brivo Event which updates MINDBODY of the client arrival. Client arrivals are cached and only updated once per day.

## Provisioning Environments 

### Setting up Brivo OnAir

+ Create a "Members" user group and add the GroupID to [.env](.env)
+ Create a Custom Field called `Barcode ID` of type `Text` and add the FieldID to [.env](.env)
+ Create a Custom Field called `User Type` of type `Text` and add the FieldID to [.env](.env)

### Create MINDBODY Webhook Subscriptions

This app requires active webhook subscriptions for:

+ client.created
+ client.updated
+ client.deactivated

See [Webhook Subscriptions](https://developers.mindbodyonline.com/WebhooksDocumentation#subscriptions) documentation.

For validation, we use the `X-Mindbody-Signature` header and the `messageSignatureKey` returned from the `POST` Subscription webhook endpoint. [Read more](https://developers.mindbodyonline.com/WebhooksDocumentation?shell#x-mindbody-signature-header)

### Create Brivo Event Subscriptions

Create Event Subscriptions for each of the Brivo sites you want the application to monitor. 

```json
{
  "name" : "Event Name",
  "url" : "https://your-application-url",
  "errorEmail": "you@error-email.com",
  "criteria": [{
            "criteriaType": "SITE",
            "criteriaOperationType": "EQ",
            "criteriaValue": 12345
        }
    ]
}
```
See [Event Subscription](https://apidocs.brivo.com/#api-Event_Subscription) documentation.

## Local Development

### Set up Redis Server

```sh
# Install Redis server
$ brew install redis
```

```sh
# Start Redis service
$ brew services start redis
```

```sh
# Connect to Redis CLI
$ redis-cli
```

### Create Environment Variables

```
# Brivo
brivo_username              [string]    Brivo OnAir username
brivo_password              [string]    Brivo OnAir password
brivo_client_id             [string]    Brivo OnAir application with password authentication type
brivo_client_secret         [string]    Same as `brivo_client_id`
brivo_api_key               [string]    Brivo developer account
brivo_facility_code         [int]       Credential facility code for member site access
brivo_site_id               [int]       GET site listing API
brivo_member_group_id       [int]       GET group listing API
brivo_barcode_field_id      [int]       GET custom field listing API
brivo_user_type_field_id    [int]       GET Custom field listing API
brivo_rate_limit            [int]       Development:20, Production:50

# Mindbody
mindbody_api_key                [string]    Mindbody developer account
mindbody_username               [string]    Mindbody username
mindbody_password               [string]    Mindbody password
mindbody_site                   [int]       Mindbody site ID (-99 for sandbox)
mindbody_location_id            [int]       GET site locations API
mindbody_message_signature_key  [string]    X-MINDBODY Signature Header

# Redis
REDIS_URL       [string]        URL of Redis server instance

# Environment
DEBUG           [bool]          Enable debug logs
PROXY           [bool]          Enable proxy debugging
PORT            [int]           Local http port for server
ENV             [string]        development | staging | production
```

### Configure Go Modules

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

### Run the Application

#### Brivo Migration Script

```sh
# On first run, migrate all MINDBODY users to Brivo
$ go run cmd/migrate/main.go
```

#### Event API Server

```sh
# Run the application and listen for webhook events
$ go run cmd/server/main.go
```

#### Clear Brivo OnAir Development Environment

```sh
# This will remove all users and credentials from Brivo
$ go run cmd/clean/main.go
```

## Heroku Integration

This application is designed to run on a basic Heroku hobby dyno. Code commits auto-deploy from `develop` to staging and `master` to production in a Heroku application pipeline.

Redis is required to cache client arrivals when the user scans into a Brivo access point. This allows us to only log one client arrival per day.

+ Install the [Heroku CLI](https://devcenter.heroku.com/articles/heroku-cli)
+ Create config vars from [.env](.env) on Heroku [link](https://devcenter.heroku.com/articles/config-vars#managing-config-vars)
+ Add Heroku Redis to the application
    + `heroku addons:create heroku-redis:hobby-dev -app your_app_name`
+ Share Redis instance with staging dyno
    + `heroku addons:attach my-originating-app::REDIS --app your_staging_app_name`
+ Get the Heroku Redis URL
    + `heroku config -a your_app_name | grep REDIS`
+ Connect to the Redis instance
    + `heroku redis:cli -a your_app_name -c your_app_name`

### Developing Locally with Heroku

```sh
# Compile the server application for Heroku
$ go build -o bin/server -v cmd/server/main.go
```

```sh
# Run server application locally
$ heroku local web
```

### Deploying to Heroku
```sh
# This isn't needed if your Heroku pipeline is configured to auto-deploy
$ git push heroku master
```

## License

See [LICENSE](LICENSE)