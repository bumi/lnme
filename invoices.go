package main

import (
	"flag"
	"github.com/bumi/lntip/ln"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
)

var stdOutLogger = log.New(os.Stdout, "", log.LstdFlags)

type TemplateRenderer struct {
	templates *template.Template
}

func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	// Add global methods if data is a map
	if viewContext, isMap := data.(map[string]interface{}); isMap {
		viewContext["reverse"] = c.Echo().Reverse
	}

	return t.templates.ExecuteTemplate(w, name, data)
}

func main() {
	var indexView = `
<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<title></title>
    <script type="text/javascript"> 
      function payme(memo, amount) {
        fetch('/payme?memo=' + memo + '&amount=' + amount)
          .then(function(response) {
            console.log(response.json());
          });
      }
    </script>
	</head>
	<body>
    hallo
	</body>
</html>`

	address := flag.String("address", "localhost:10009", "The host and port of the ln gRPC server")
	certFile := flag.String("cert", "tls.cert", "Path to the lnd tls.cert file")
	macaroonFile := flag.String("macaroon", "invoice.macaroon", "Path to the lnd macaroon file")
	viewPath := flag.String("template", "", "Path of a custom HTML template file")

	flag.Parse()

	if *viewPath != "" {
		content, err := ioutil.ReadFile(*viewPath)
		if err != nil {
			panic(err)
		}
		indexView = string(content)
	}

	e := echo.New()
	e.Static("/static", "views/assets")
	e.Use(middleware.CORS())
	e.Use(middleware.Recover())
	renderer := &TemplateRenderer{
		templates: template.Must(template.New("index").Parse(indexView)),
	}
	e.Renderer = renderer

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

	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "index", map[string]interface{}{})
	})

	e.Logger.Fatal(e.Start(":1323"))
}
