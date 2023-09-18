package main

import (
    "context"
    "crypto/tls"
    "crypto/x509"
    "os"

    "github.com/elastic/go-lumber/server"

    "github.com/benthosdev/benthos/v4/public/service"

    // Import specific Benthos components
    _ "github.com/benthosdev/benthos/v4/public/components/crypto"
    _ "github.com/benthosdev/benthos/v4/public/components/io"
)

// struct of the input configuration fields
type LumberJackInput struct {
    bind    string // address and port to use for the listener
    svrCert string // Server CA-signed certificate
    privKey string // The server svc private key
    caCert  string // CA root cert for authenticating client cert auth
    cliAuth bool   // Whether to enforce client certificate auth

    ljServer server.Server // the lumberjack server
}

//-----------------------------------------------------------------------------------
// Lumberjack service BatchInput interface methods - changed from Input to BatchInput
//-----------------------------------------------------------------------------------

func (l *LumberJackInput) Connect(ctx context.Context) error {
    // This is the method for connecting to the upstream Lumberjack server service that listens
    // for client connections. Connect is starting the Lumberjack Server service.

    // Create the TLS Config for the service
    tlsConfig := &tls.Config{}

    // If Client Cert Auth enabled, configure tls.Config to enforce and provide CA cert
    if l.cliAuth {
   	 caCert, err := os.ReadFile(l.caCert)
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

    ls, err := server.ListenAndServe(l.bind,
   	 server.V1(false),
   	 server.V2(true),
   	 server.TLS(tlsConfig),
    )
    if err != nil {
   	 return err
    }

    l.ljServer = ls

    return nil
}

func (l *LumberJackInput) ReadBatch(ctx context.Context) (service.MessageBatch, service.AckFunc, error) {
    batchChan := l.ljServer.ReceiveChan()
    select {
    case lJBatch := <-batchChan: // read a batch from the receiver channel
   	 lJBatch.ACK() // Ack the received batch
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
    case <-ctx.Done():
   	 return nil, nil, ctx.Err()
    }
}

func (l *LumberJackInput) Close(ctx context.Context) error {
    l.ljServer.Close()
    return nil
}

//------------------------------------------------------------------------------

func main() {
    // Lumberjack Input Config Spec
    ljConfigSpec := service.NewConfigSpec().
   	 Summary("Creates a Lumberjack protocol server listener").
   	 Field(service.NewStringField("bind").
   		 Default("0.0.0.0:5044").
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

    // Lumberjack Input constructor
    ljConstructor := func(conf *service.ParsedConfig, mgr *service.Resources) (service.BatchInput, error) {
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
   	 return service.AutoRetryNacksBatched(&LumberJackInput{bind, svrCert, privKey, caCert,
   		 cliAuth, ljserver}), nil
    }
    // Register Lumberjack Input
    err := service.RegisterBatchInput("lumberjack", ljConfigSpec, ljConstructor)
    if err != nil {
   	 panic(err)
    }

    service.RunCLI(context.Background())
}
