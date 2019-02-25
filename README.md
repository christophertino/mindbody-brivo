# FIAO Brooklyn Membership API

Sync membership data between MindBody and Brivo Access APIs. Built for the Federation of Italian-American Organizations of Brooklyn. 

## Generating API Keys

+ [Brivo OnAir API](https://developer.brivo.com/)
+ [MindBody](https://developers.mindbodyonline.com/)

## Running the App

```sh
# If running inside $GOPATH
$ export GO111MODULE=on
```

```sh
# Install dependencies with Go modules
$ go mod init
$ go build cmd/fiao_api/main.go
```

```sh
# Generate configuration file
$ cp conf/conf.example.json conf/conf.json
```

```sh
$ go run cmd/fiao_api/main.go
```

## Credits

+ [Mindbody-API-Golang](https://github.com/vacovsky/Mindbody-API-Golang)