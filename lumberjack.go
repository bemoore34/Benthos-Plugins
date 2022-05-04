package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"os"
	"os/signal"

	"github.com/benthosdev/benthos/v4/public/service"
	"github.com/elastic/go-lumber/server"

	// Import all standard Benthos components
	_ "github.com/benthosdev/benthos/v4/public/components/all"
)

// Input configuration fields
type lumberJackInput struct {
	bind    string // address and port to use for the listener
	svrCert string // Server CA-signed certificate
	privKey string // The server svc private key
	caCert  string // CA root cert for authenticating client cert auth
	cliAuth bool   // Whether to enforce client certificate auth

	ljServer server.Server  // the lumberjack server
	sig      chan os.Signal // used for signaling the lj server to terminate
}

//-----------------------------------------------------------------------------
// service.BatchInput interface methods: Connect, ReadBatch, Close
//-----------------------------------------------------------------------------

// Creates the lumberjack server
func (l *lumberJackInput) Connect(ctx context.Context) error {

  // Create the TLS Config for the service - TLS is enabled in this verion (not optional)
	tlsConfig := &tls.Config{}

	// If Client Cert Auth enabled, configure tls.Config to enforce and provide CA cert
	if l.cliAuth {
		caCert, err := ioutil.ReadFile(l.caCert)
		if err != nil {
			return err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.ClientCAs = caCertPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	keycert, err := tls.LoadX509KeyPair(l.svrCert, l.privKey)
	if err != nil {
		return err
	}
	tlsConfig.Certificates = append(tlsConfig.Certificates, keycert)

	// Start the LJ server - V2 with TLS enabled.
  ls, err := server.ListenAndServe(l.bind,
		server.V1(false),
		server.V2(true),
		server.TLS(tlsConfig),
	)
	if err != nil {
		return err
	}
  
	// signal channel to capture an interrupt (ctrl-c) to gracefully stop the server.
	l.sig = make(chan os.Signal, 1)
	signal.Notify(l.sig, os.Interrupt)
	go func() {
		<-l.sig
		_ = ls.Close()
		os.Exit(0)
	}()
	l.ljServer = ls
	return nil
}

// The LJ server returns batches of messages
func (l *lumberJackInput) ReadBatch(ctx context.Context) (service.MessageBatch, service.AckFunc, error) {
	batchChan := l.ljServer.ReceiveChan()
	// Read a batch from the LJ Batch Receive Channel
	lJBatch := <-batchChan
	lJBatch.ACK()
	
  // Convert the ljBatch to a Benthos service.MessageBatch
	var batch service.MessageBatch
	for _, v := range lJBatch.Events {
		msg := service.NewMessage(nil)
		msg.SetStructured(v)
		batch = append(batch, msg)
	}
	return batch, func(ctx context.Context, err error) error {
		return nil
	}, nil
}

// I don't know if I should be using this - couldn't get it to stop the server, so using the 
// signal channel in the Connect method.
func (l *lumberJackInput) Close(ctx context.Context) error {
	//_ = l.ljServer.Close()
	//os.Exit(0)
	return nil
}

//-----------------------------------------------------------------------------

func main() {
	configSpec := service.NewConfigSpec().
		Summary("Creates a Lumberjack protocol server listener").
		Field(service.NewStringField("bind").
			Default(`":5044"`).
			Description("The Listening IP address and port. E.g. 192.168.0.1:9099")).
		Field(service.NewStringField("svrCert").
			Description("The signed cert for the TLS-enabled server service")).
		Field(service.NewStringField("privKey").
			Description("The private key for the TLS-enabled server service")).
		Field(service.NewStringField("caCert").
			Description("The CA Public certificate for client connections").Optional()).
		Field(service.NewBoolField("cliAuth").
			Default(false).
			Description("Enforce Client certificate auth."))
	
  // Input constructor
	constructor := func(conf *service.ParsedConfig, mgr *service.Resources) (service.BatchInput, error) {
		bind, err := conf.FieldString("bind")
		if err != nil {
			return nil, err
		}
		svrCert, err := conf.FieldString("svrCert")
		if err != nil {
			return nil, err
		}
		privKey, err := conf.FieldString("privKey")
		if err != nil {
			return nil, err
		}
		caCert, err := conf.FieldString("caCert")
		if err != nil {
			return nil, err
		}
		cliAuth, err := conf.FieldBool("cliAuth")
		if err != nil {
			return nil, err
		}
		var ljserver server.Server
		var sig chan os.Signal
		return service.AutoRetryNacksBatched(&lumberJackInput{bind, svrCert, privKey, caCert,
			cliAuth, ljserver, sig}), nil
	}
	
  err := service.RegisterBatchInput("lumberjack", configSpec, constructor)
	if err != nil {
		panic(err)
	}
  
	service.RunCLI(context.Background())
}
