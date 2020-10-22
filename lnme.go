package main

import (
	"flag"
	"github.com/GeertJohan/go.rice"
	"github.com/bumi/lntip/ln"
	"github.com/didip/tollbooth/v6"
	"github.com/didip/tollbooth/v6/limiter"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"log"
	"net/http"
	"os"
)

// move to file
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
	address := flag.String("address", "localhost:10009", "The host and port of the ln gRPC server")
	certFile := flag.String("cert", "~/.lnd/tls.cert", "Path to the lnd tls.cert file")
	macaroonFile := flag.String("macaroon", "~/.lnd/data/chain/bitcoin/mainnet/invoice.macaroon", "Path to the lnd macaroon file")
	bind := flag.String("bind", ":1323", "Host and port to bind on")
	staticPath := flag.String("static-path", "", "Path to a static assets directory. Blank to not serve any static files")
	disableHTML := flag.Bool("disable-html", false, "Disable HTML page")
	disableCors := flag.Bool("disable-cors", false, "Disable CORS headers")
	requestLimit := flag.Float64("request-limit", 5, "Request limit per second")

	flag.Parse()

	e := echo.New()
	if *staticPath != "" {
		e.Static("/", *staticPath)
	} else if !*disableHTML {
		rootBox := rice.MustFindBox("files/root")
		indexPage, err := rootBox.String("index.html")
		if err == nil {
			stdOutLogger.Print("Running page")
			e.GET("/", func(c echo.Context) error {
				return c.HTML(200, indexPage)
			})
		}
	}

	if !*disableCors {
		e.Use(middleware.CORS())
	}
	e.Use(middleware.Recover())

	if *requestLimit > 0 {
		limiter := tollbooth.NewLimiter(*requestLimit, nil)
		e.Use(LimitMiddleware(limiter))
	}

	// Embed static files and serve those on /lnme (e.g. /lnme/lnme.js)
	assetHandler := http.FileServer(rice.MustFindBox("files/assets").HTTPBox())
	e.GET("/lnme/*", echo.WrapHandler(http.StripPrefix("/lnme/", assetHandler)))

	stdOutLogger.Printf("Connection to %s using macaroon %s and cert %s", *address, *macaroonFile, *certFile)
	lndOptions := ln.LNDoptions{
		Address:      *address,
		CertFile:     *certFile,
		MacaroonFile: *macaroonFile,
	}
	lnClient, err := ln.NewLNDclient(lndOptions)
	if err != nil {
		stdOutLogger.Print("Error initializing LND client:")
		panic(err)
	}

	// endpoint URLs compatible to the LND REST API
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

  e.POST("/v1/newaddress", func(c echo.Context) error {
    address, err := lnClient.NewAddress()
    if err != nil {
			stdOutLogger.Printf("Error getting a new BTC address: %s", err)
			return c.JSON(http.StatusInternalServerError, "Error getting address")
    }
    return c.JSON(http.StatusOK, address)
  })

	e.GET("/v1/invoice/:invoiceId", func(c echo.Context) error {
		invoiceId := c.Param("invoiceId")
		invoice, err := lnClient.GetInvoice(invoiceId)

		if err != nil {
			stdOutLogger.Printf("Error looking up invoice: %s", err)
			return c.JSON(http.StatusInternalServerError, "Error fetching invoice")
		}

		return c.JSON(http.StatusOK, invoice)
	})

	// debug test endpoint
	e.GET("/ping", func(c echo.Context) error {
		return c.JSON(http.StatusOK, "pong")
	})

	e.Logger.Fatal(e.Start(*bind))
}
