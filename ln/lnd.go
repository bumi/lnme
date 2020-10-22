package ln

import (
	"context"
	"encoding/hex"
	"io/ioutil"
	"log"
	"os"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/macaroons"
	"gopkg.in/macaroon.v2"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var stdOutLogger = log.New(os.Stdout, "", log.LstdFlags)

// thanks https://github.com/philippgille/ln-paywall/
// Invoice is a Lightning Network invoice and contains the typical invoice string and the payment hash.
type Invoice struct {
	PaymentHash    string `json:"payment_hash"`
	PaymentRequest string `json:"payment_request"`
	Settled        bool   `json:"settled"`
}

type LNDclient struct {
	lndClient lnrpc.LightningClient
	ctx       context.Context
	conn      *grpc.ClientConn
}

// AddInvoice generates an invoice with the given price and memo.
func (c LNDclient) AddInvoice(value int64, memo string) (Invoice, error) {
	result := Invoice{}

	stdOutLogger.Printf("Adding invoice: memo=%s amount=%v ", memo, value)
	invoice := lnrpc.Invoice{
		Memo:  memo,
		Value: value,
	}
	res, err := c.lndClient.AddInvoice(c.ctx, &invoice)
	if err != nil {
		return result, err
	}

	result.PaymentHash = hex.EncodeToString(res.RHash)
	result.PaymentRequest = res.PaymentRequest
	return result, nil
}

func (c LNDclient) NewAddress() (string, error) {
	stdOutLogger.Printf("Getting a new BTC address")
  request := lnrpc.NewAddressRequest{
    Type: lnrpc.AddressType_WITNESS_PUBKEY_HASH,
  }
  res, err := c.lndClient.NewAddress(c.ctx, &request)
  if err != nil {
    return "", err
  }
  return res.Address, nil
}

// GetInvoice takes an invoice ID and returns the invoice details including settlement details
// An error is returned if no corresponding invoice was found.
func (c LNDclient) GetInvoice(paymentHashStr string) (Invoice, error) {
	var invoice Invoice
	stdOutLogger.Printf("Getting invoice: hash=%s\n", paymentHashStr)

	plainHash, err := hex.DecodeString(paymentHashStr)
	if err != nil {
		return invoice, err
	}

	// Get the invoice for that hash
	paymentHash := lnrpc.PaymentHash{
		RHash: plainHash,
		// Hex encoded, must be exactly 32 byte
		RHashStr: paymentHashStr,
	}
	result, err := c.lndClient.LookupInvoice(c.ctx, &paymentHash)
	if err != nil {
		return invoice, err
	}

	invoice = Invoice{}
	invoice.PaymentHash = hex.EncodeToString(result.RHash)
	invoice.PaymentRequest = result.PaymentRequest
	invoice.Settled = result.GetSettled()

	return invoice, nil
}

func NewLNDclient(lndOptions LNDoptions) (LNDclient, error) {
	result := LNDclient{}

	creds, err := credentials.NewClientTLSFromFile(lndOptions.CertFile, "")
	if err != nil {
		return result, err
	}
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
	}

	macaroonData, err := ioutil.ReadFile(lndOptions.MacaroonFile)
	if err != nil {
		return result, err
	}
	mac := &macaroon.Macaroon{}
	if err = mac.UnmarshalBinary(macaroonData); err != nil {
		return result, err
	}

	macCred := macaroons.NewMacaroonCredential(mac)
	opts = append(opts, grpc.WithPerRPCCredentials(macCred))

	conn, err := grpc.Dial(lndOptions.Address, opts...)
	if err != nil {
		return result, err
	}

	c := lnrpc.NewLightningClient(conn)

	result = LNDclient{
		conn:      conn,
		ctx:       context.Background(),
		lndClient: c,
	}

	return result, nil
}

// LNDoptions are the options for the connection to the lnd node.
type LNDoptions struct {
	// Address of your LND node, including the port.
	// Optional ("localhost:10009" by default).
	Address string
	// Path to the "tls.cert" file that your LND node uses.
	// Optional ("tls.cert" by default).
	CertFile string
	// Path to the macaroon file that your LND node uses.
	// "invoice.macaroon" if you only use the AddInvoice() and GetInvoice() methods
	// (required by the middleware in the package "wall").
	// "admin.macaroon" if you use the Pay() method (required by the client in the package "pay").
	// Optional ("invoice.macaroon" by default).
	MacaroonFile string
}
