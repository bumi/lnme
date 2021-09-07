// +heroku goVersion 1.15
module github.com/bumi/lnme

go 1.15

// https://github.com/lightningnetwork/lnd/issues/5624#issuecomment-897512230
replace go.etcd.io/etcd => go.etcd.io/etcd v0.5.0-alpha.5.0.20201125193152-8a03d2e9614b
require (
	github.com/GeertJohan/go.rice v1.0.2
	github.com/cretz/bine v0.2.0
	github.com/didip/tollbooth/v6 v6.1.1
	github.com/knadh/koanf v1.2.1
	github.com/labstack/echo/v4 v4.5.0
	github.com/lightningnetwork/lnd v0.13.1-beta
	google.golang.org/grpc v1.40.0
	gopkg.in/macaroon.v2 v2.1.0
)
