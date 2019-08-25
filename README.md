# LnTip - your friendly lightning tipping widget

LnTip provides a Bitcoin lightning tipping widget that can easily be integrated into any website.  

It consistes of a small service written in Go that connects to a lnd node and exposes 
a simple HTTP JSON API to create and monitor invoices. This is a HTTP/REST proxy to the LND add and receive invoices API.

That API is consumed from a tiny JavaScript widget that can be integrated into any website. 

If [webln](https://github.com/wbobeirne/webln) is available the widget automatically use webln to request the payment; 
otherwise an overlay will be shown with the payment request and a QR code.


## Motivation

I wanted a simple tipping button for my website that uses my own lightning node and does not rely on external services (does not need to trusts external services to handle the payments and hold the coins).


## Installation

To use LnTip a running [LND node](https://github.com/lightningnetwork/lnd/blob/master/docs/INSTALL.md) 
is required.  

1. download the latest [release](https://github.com/bumi/lntip/releases)
2. run `invoices_proxy` (to run it as systemd service have a look at the [systemd service example config](https://github.com/bumi/lntip/blob/master/examples/invoices-proxy.service))
3. integrate the widget on website

### Configuration

To connect to the lnd node the cert, macaroon and address of the lnd node has to be configured:

* address: Host and port of the lnd gRPC service. default: localhost:10009
* cert: Path to the lnd cert file. default: ~/.lnd/tls.cert
* macaroon: Path to the macaroon file. default: ~/.lnd/data/chain/bitcoin/mainnet/invoice.macaroon
* bind: Host and port to listen on. default: :1323 (localhost:1323)
* static-path: The proxy can serve files from a static folder (e.g. the JS/CSS files). Use this option to configure the path to a filder. (e.g. /home/bitcoin/lntip/assets) default: disabled
* disable-cors: Disable CORS headers. (default: false)
* request-limit: Limit the allowed requests per second. (default: 10)

Examples: 

    $ ./invoices_proxy --help
    $ ./invoices_proxy --address=lndhost.com:10009 --bind=localhost:4711


### JavaScript Widget integration

Load the JavaScript file in your HTML page and configure the `lntip-host` attribute 
to the host and port on which your lntip instance is running:

```html
<script lntip-host="https://your-lntip-host.com:1323" src="https://cdn.jsdelivr.net/gh/bumi/lntip/assets/lntip.js" id="lntip-script"></script>
```

#### Usage

To request a lightning payment simply call `request()` on a `new LnTip({value: value, memo: memo})`:

```js
new LnTip({ value: 1000, memo: 'high5' }).request()
```

Use it from a plain HTML link:
```html
  <a href="#" onclick="javascript:new LnTip({ value: 1000, memo: 'high5' }).request();return false;">Tip me</a>
```

##### More advanced JS API:

```js
let tip = new LnTip({ value: 1000, memo: 'high5' });

// get a new invoice and watch for a payment
// promise resolves if the invoice is settled
tip.requestPayment().then((invoice) => {
  alert('YAY, thanks!');
});

// create a new invoice
tip.addInvoice().then((invoice) => {
  console.log(invoice.PaymentRequest)
});

// periodically watch if an invoice is settled
tip.watchPayment().then((invoice) => {
  alert('YAY, thanks!');
});

```

## Development

Use `go run` to ron the service locally: 

    $ go run invoices_proxy.go --address=127.0.0.1:10009 --cert=/home/bitcoin/lightning/tls.cert --macaroon=/home/bitcoin/lightning/invoice.macaroon


## Contributing

Bug reports and pull requests are welcome on GitHub at https://github.com/bumi/lntip

## License

Available as open source under the terms of the [MIT License](http://opensource.org/licenses/MIT).
