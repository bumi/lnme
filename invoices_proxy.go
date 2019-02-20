package main

import (
	"flag"
	"github.com/bumi/lntip/ln"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"log"
	"net/http"
	"os"
)

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
	staticPath := flag.String("static-path", "", "Path to a static assets directory. Blank to disable serving static files")
	disableCors := flag.Bool("disable-cors", false, "Disable CORS headers")

	flag.Parse()

	e := echo.New()
	if *staticPath != "" {
		e.Static("/static", *staticPath)
	}
	if !*disableCors {
		e.Use(middleware.CORS())
	}
	e.Use(middleware.Recover())

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
			return c.JSON(http.StatusBadRequest, "bad request")
		}

		invoice, err := lnClient.AddInvoice(i.Value, i.Memo)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, "invoice creation error")
		}

		return c.JSON(http.StatusOK, invoice)
	})

	e.GET("/v1/invoice/:invoiceId", func(c echo.Context) error {
		invoiceId := c.Param("invoiceId")
		invoice, _ := lnClient.GetInvoice(invoiceId)
		return c.JSON(http.StatusOK, invoice)
	})

	e.Logger.Fatal(e.Start(*bind))
}
