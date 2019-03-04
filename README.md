# FIAO Brooklyn Membership Sync

Sync membership data between MindBody and Brivo OnAir. Built for the Federation of Italian-American Organizations of Brooklyn. 

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
$ go run cmd/fiao/main.go
```

## License
[MPL-2.0](https://www.mozilla.org/en-US/MPL/2.0/) Copyright 2019 FIAO Brooklyn. All rights reserved.

See [LICENSE](LICENSE)