package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"strings"

	rice "github.com/GeertJohan/go.rice"
	"github.com/bumi/lnme/ln"
	"github.com/didip/tollbooth/v6"
	"github.com/didip/tollbooth/v6/limiter"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/basicflag"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Middleware for request limited to prevent too many requests
// TODO: move to file
func LimitMiddleware(lmt *limiter.Limiter) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return echo.HandlerFunc(func(c echo.Context) error {
			httpError := tollbooth.LimitByRequest(lmt, c.Response(), c.Request())
			if httpError != nil {
				return c.String(httpError.StatusCode, httpError.Message)
			}
			return next(c)
		})
	}
}

var stdOutLogger = log.New(os.Stdout, "", log.LstdFlags)

type Invoice struct {
	Value int64  `json:"value"`
	Memo  string `json:"memo"`
}

func main() {
	cfg := LoadConfig()

	e := echo.New()

	// Serve static files if configured
	if cfg.String("static-path") != "" {
		e.Static("/", cfg.String("static-path"))
		// Serve default page
	} else if !cfg.Bool("disable-website") {
		rootBox := rice.MustFindBox("files/root")
		indexPage, err := rootBox.String("index.html")
		if err == nil {
			stdOutLogger.Print("Running embedded page")
			e.GET("/", func(c echo.Context) error {
				return c.HTML(200, indexPage)
			})
		} else {
			stdOutLogger.Printf("Failed to run embedded website: %s", err)
		}
	}
	// Embed static files and serve those on /lnme (e.g. /lnme/lnme.js)
	assetHandler := http.FileServer(rice.MustFindBox("files/assets").HTTPBox())
	e.GET("/lnme/*", echo.WrapHandler(http.StripPrefix("/lnme/", assetHandler)))

	// CORS settings
	if !cfg.Bool("disable-cors") {
		e.Use(middleware.CORS())
	}

	// Recover middleware recovers from panics anywhere in the request chain
	e.Use(middleware.Recover())

	// Request limit per second. DoS protection
	if cfg.Int("request-limit") > 0 {
		limiter := tollbooth.NewLimiter(cfg.Float64("request-limit"), nil)
		e.Use(LimitMiddleware(limiter))
	}

	// Setup lightning client
	stdOutLogger.Printf("Connecting to %s", cfg.String("lnd-address"))
	lndOptions := ln.LNDoptions{
		Address:      cfg.String("lnd-address"),
		CertFile:     cfg.String("lnd-cert-path"),
		CertHex:      cfg.String("lnd-cert"),
		MacaroonFile: cfg.String("lnd-macaroon-path"),
		MacaroonHex:  cfg.String("lnd-macaroon"),
	}
	lnClient, err := ln.NewLNDclient(lndOptions)
	if err != nil {
		stdOutLogger.Print("Error initializing LND client:")
		panic(err)
	}

	// Endpoint URLs compatible to the LND REST API v1
	//
	// Create new invoice
	e.POST("/v1/invoices", func(c echo.Context) error {
		i := new(Invoice)
		if err := c.Bind(i); err != nil {
			stdOutLogger.Printf("Bad request: %s", err)
			return c.JSON(http.StatusBadRequest, "Bad request")
		}

		invoice, err := lnClient.AddInvoice(i.Value, i.Memo)
		if err != nil {
			stdOutLogger.Printf("Error creating invoice: %s", err)
			return c.JSON(http.StatusInternalServerError, "Error adding invoice")
		}

		return c.JSON(http.StatusOK, invoice)
	})

	// Get next BTC onchain address
	e.POST("/v1/newaddress", func(c echo.Context) error {
		address, err := lnClient.NewAddress()
		if err != nil {
			stdOutLogger.Printf("Error getting a new BTC address: %s", err)
			return c.JSON(http.StatusInternalServerError, "Error getting address")
		}
		return c.JSON(http.StatusOK, address)
	})

	// Check invoice status
	e.GET("/v1/invoice/:paymentHash", func(c echo.Context) error {
		paymentHash := c.Param("paymentHash")
		invoice, err := lnClient.GetInvoice(paymentHash)

		if err != nil {
			stdOutLogger.Printf("Error looking up invoice: %s", err)
			return c.JSON(http.StatusInternalServerError, "Error fetching invoice")
		}

		return c.JSON(http.StatusOK, invoice)
	})

	// Debug test endpoint
	e.GET("/ping", func(c echo.Context) error {
		return c.JSON(http.StatusOK, "pong")
	})

	port := cfg.String("port")
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}
	e.Logger.Fatal(e.Start(":" + port))
}

func LoadConfig() *koanf.Koanf {
	k := koanf.New(".")

	f := flag.NewFlagSet("LnMe", flag.ExitOnError)
	f.String("lnd-address", "localhost:10009", "The host and port of the LND gRPC server.")
	f.String("lnd-macaroon-path", "~/.lnd/data/chain/bitcoin/mainnet/invoice.macaroon", "Path to the LND macaroon file.")
	f.String("lnd-cert-path", "~/.lnd/tls.cert", "Path to the LND tls.cert file.")
	f.Bool("disable-website", false, "Disable default embedded website.")
	f.Bool("disable-cors", false, "Disable CORS headers.")
	f.Float64("request-limit", 5, "Request limit per second.")
	f.String("static-path", "", "Path to a static assets directory.")
	f.String("port", "1323", "Port to bind on.")
	var configPath string
	f.StringVar(&configPath, "config", "config.toml", "Path to a .toml config file.")
	f.Parse(os.Args[1:])

	// Load config from flags, including defaults
	if err := k.Load(basicflag.Provider(f, "."), nil); err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Load config from environment variables
	k.Load(env.Provider("LNME_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(strings.TrimPrefix(s, "LNME_")), "_", "-", -1)
	}), nil)

	// Load config from file if available
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			if err := k.Load(file.Provider(configPath), toml.Parser()); err != nil {
				log.Fatalf("Error loading config file: %v", err)
			}
		}
	}

	return k
}
