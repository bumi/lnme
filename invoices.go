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
	Amount int64  `json:"amount"`
	Memo   string `json:"memo"`
}

func main() {
	address := flag.String("address", "localhost:10009", "The host and port of the ln gRPC server")
	certFile := flag.String("cert", "tls.cert", "Path to the lnd tls.cert file")
	macaroonFile := flag.String("macaroon", "invoice.macaroon", "Path to the lnd macaroon file")
	bind := flag.String("bind", ":1323", "Host and port to bind on")

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

	e.POST("/invoice", func(c echo.Context) error {
		i := new(Invoice)
		if err := c.Bind(i); err != nil {
			return c.JSON(http.StatusBadRequest, "bad request")
		}

		invoice, err := lnClient.GenerateInvoice(i.Amount, i.Memo)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, "invoice creation error")
		}

		return c.JSON(http.StatusOK, invoice)
	})

	e.GET("/settled/:invoiceId", func(c echo.Context) error {
		invoiceId := c.Param("invoiceId")
		invoice, _ := lnClient.CheckInvoice(invoiceId)
		return c.JSON(http.StatusOK, invoice)
	})

	e.Logger.Fatal(e.Start(*bind))
}
