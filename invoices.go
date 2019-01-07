package main

import (
	"flag"
	"github.com/bumi/lntip/ln"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"log"
	"net/http"
	"os"
	"strconv"
)

var stdOutLogger = log.New(os.Stdout, "", log.LstdFlags)

func main() {
	address := flag.String("address", "localhost:10009", "The host and port of the ln gRPC server")
	certFile := flag.String("cert", "tls.cert", "Path to the lnd tls.cert file")
	macaroonFile := flag.String("macaroon", "invoice.macaroon", "Path to the lnd macaroon file")

	flag.Parse()

	e := echo.New()
	e.Static("/static", "assets")
	e.Use(middleware.CORS())
	e.Use(middleware.Recover())

  lndOptions := ln.LNDoptions{
		Address:      *address,
		CertFile:     *certFile,
		MacaroonFile: *macaroonFile,
	}
	lnClient, err := ln.NewLNDclient(lndOptions)
	if err != nil {
		panic(err)
	}

	e.GET("/payme", func(c echo.Context) error {
		memo := c.FormValue("memo")
		amount, _ := strconv.ParseInt(c.FormValue("amount"), 10, 64)
		invoice, err := lnClient.GenerateInvoice(amount, memo)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, "error")
		}

		return c.JSON(http.StatusOK, invoice)
	})

	e.GET("/settled/:invoiceId", func(c echo.Context) error {
		invoiceId := c.Param("invoiceId")
		invoice, _ := lnClient.CheckInvoice(invoiceId)
		return c.JSON(http.StatusOK, invoice)
	})

	e.Logger.Fatal(e.Start(":1323"))
}
