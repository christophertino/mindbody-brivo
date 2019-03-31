# FIAO Brooklyn Membership Sync

Sync membership data between MindBody and Brivo OnAir. This application makes the following assumptions:

+ Master client list is stored in MindBody and mirrored to Brivo
+ Mindbody users have been assigned a wristband with a generated barcode ID
+ Brivo credentials are cleard when the user is deactivated in MindBody

Built for the Federation of Italian-American Organizations of Brooklyn. 

## Generating API Keys

+ [Brivo OnAir API](https://developer.brivo.com/)
+ [MindBody API](https://developers.mindbodyonline.com/)

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
# Run the application. Use optional argument 'provision' for first-run
$ go run cmd/fiao/main.go [provision]
```

## License
[MPL-2.0](https://www.mozilla.org/en-US/MPL/2.0/) Copyright 2019 FIAO Brooklyn. All rights reserved.

See [LICENSE](LICENSE)