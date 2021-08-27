package ln

import (
	"context"
	"crypto/x509"
	"encoding/hex"
	"fmt"
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

// LNDoptions are the options for the connection to the lnd node.
type LNDoptions struct {
	Address      string
	CertFile     string
	CertHex      string
	MacaroonFile string
	MacaroonHex  string
}

type LNDclient struct {
	lndClient lnrpc.LightningClient
	ctx       context.Context
	conn      *grpc.ClientConn
}

// AddInvoice generates an invoice with the given price and memo.
func (c LNDclient) AddInvoice(value int64, memo string) (Invoice, error) {
	result := Invoice{}

	stdOutLogger.Printf("Adding invoice: memo=%s value=%v ", memo, value)
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

// NewAddress gets the next BTC onchain address.
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

	// Get credentials either from a hex string or a file
	var creds credentials.TransportCredentials
	// if a hex string is provided
	if lndOptions.CertHex != "" {
		cp := x509.NewCertPool()
		cert, err := hex.DecodeString(lndOptions.CertHex)
		if err != nil {
			return result, err
		}
		cp.AppendCertsFromPEM(cert)
		creds = credentials.NewClientTLSFromCert(cp, "")
		// if a path to a cert file is provided
	} else if lndOptions.CertFile != "" {
		credsFromFile, err := credentials.NewClientTLSFromFile(lndOptions.CertFile, "")
		if err != nil {
			return result, err
		}
		creds = credsFromFile // make it available outside of the else if block
	} else {
		return result, fmt.Errorf("LND credential is missing")
	}
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
	}

	var macaroonData []byte
	if lndOptions.MacaroonHex != "" {
		macBytes, err := hex.DecodeString(lndOptions.MacaroonHex)
		if err != nil {
			return result, err
		}
		macaroonData = macBytes
	} else if lndOptions.MacaroonFile != "" {
		macBytes, err := ioutil.ReadFile(lndOptions.MacaroonFile)
		if err != nil {
			return result, err
		}
		macaroonData = macBytes // make it available outside of the else if block
	} else {
		return result, fmt.Errorf("LND macaroon is missing")
	}

	mac := &macaroon.Macaroon{}
	if err := mac.UnmarshalBinary(macaroonData); err != nil {
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
