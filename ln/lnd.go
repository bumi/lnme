package ln

import (
	"context"
	"encoding/hex"
	"io/ioutil"
  "log"
  "os"

  "github.com/lightningnetwork/lnd/lnrpc"
  "gopkg.in/macaroon.v2"
  "github.com/lightningnetwork/lnd/macaroons"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)


var stdOutLogger = log.New(os.Stdout, "", log.LstdFlags)

// thanks https://github.com/philippgille/ln-paywall/
// Invoice is a Lightning Network invoice and contains the typical invoice string and the payment hash.
type Invoice struct {
	ImplDepID string
	PaymentHash string
	PaymentRequest string
}

type LNDclient struct {
	lndClient lnrpc.LightningClient
	ctx       context.Context
	conn      *grpc.ClientConn
}

// GenerateInvoice generates an invoice with the given price and memo.
func (c LNDclient) GenerateInvoice(amount int64, memo string) (Invoice, error) {
	result := Invoice{}

  stdOutLogger.Printf("Creating invoice: memo=%s amount=%v ", memo, amount)
	invoice := lnrpc.Invoice{
		Memo:  memo,
		Value: amount,
	}
	res, err := c.lndClient.AddInvoice(c.ctx, &invoice)
	if err != nil {
		return result, err
	}

	result.ImplDepID = hex.EncodeToString(res.RHash)
	result.PaymentHash = result.ImplDepID
	result.PaymentRequest = res.PaymentRequest
	return result, nil
}

// CheckInvoice takes an invoice ID and checks if the corresponding invoice was settled.
// An error is returned if no corresponding invoice was found.
// False is returned if the invoice isn't settled.
func (c LNDclient) CheckInvoice(id string) (bool, error) {
	// In the case of lnd, the ID is the hex encoded preimage hash.
	plainHash, err := hex.DecodeString(id)
	if err != nil {
		return false, err
	}

  stdOutLogger.Printf("Lookup invoice: hash=%s\n", id)

	// Get the invoice for that hash
	paymentHash := lnrpc.PaymentHash{
		RHash: plainHash,
		// Hex encoded, must be exactly 32 byte
		RHashStr: id,
	}
	invoice, err := c.lndClient.LookupInvoice(c.ctx, &paymentHash)
	if err != nil {
		return false, err
	}

	// Check if invoice was settled
	if !invoice.GetSettled() {
		return false, nil
	}
	return true, nil
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
	// "invoice.macaroon" if you only use the GenerateInvoice() and CheckInvoice() methods
	// (required by the middleware in the package "wall").
	// "admin.macaroon" if you use the Pay() method (required by the client in the package "pay").
	// Optional ("invoice.macaroon" by default).
	MacaroonFile string
}

