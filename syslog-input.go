package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"

	"gopkg.in/mcuadros/go-syslog.v2"

	"github.com/benthosdev/benthos/v4/public/service"

	// Import all standard Benthos components
	_ "github.com/benthosdev/benthos/v4/public/components/all"
)

// struct of the input configuration fields
type SyslogSvrInput struct {
	bind     string // address and port to use for the listener
	rfc      string // the rfc to use: RFC3164, RFC5424, RFC6587 or Automatic
	protocol string // udp or tcp
	svrCert  string // Server CA-signed certificate
	privKey  string // The server svc private key
	caCert   string // CA root cert for authenticating client cert auth
	useTLS   bool   // Whether to use TLS encryption
	cliAuth  bool   // Whether to enforce client certificate auth

	sls     *syslog.Server
	slschan *syslog.LogPartsChannel
}

//------------------------------------------------------------------------------
// Syslog server Input interface methods
//------------------------------------------------------------------------------
func (s *SyslogSvrInput) Connect(ctx context.Context) error {
	channel := make(syslog.LogPartsChannel)
	s.slschan = &channel
	handler := syslog.NewChannelHandler(channel)

	server := syslog.NewServer()
	switch s.rfc {
	case "Automatic":
		server.SetFormat(syslog.Automatic)
	case "RFC3164":
		server.SetFormat(syslog.RFC3164)
	case "RFC5424":
		server.SetFormat(syslog.RFC5424)
	case "RFC6587":
		server.SetFormat(syslog.RFC6587)
	default:
		server.SetFormat(syslog.Automatic)
	}

	server.SetHandler(handler)

	if s.protocol == "UDP" {
		server.ListenUDP(s.bind)
	} else if s.protocol == "TCP" && !s.useTLS {
		server.ListenTCP(s.bind)
	} else if s.protocol == "TCP" && s.useTLS {
		// Use TLS
		// Create the TLS Config for the service
		tlsConfig := &tls.Config{}

		// If Client Cert Auth enabled, configure tls.Config to enforce and provide CA cert
		if s.cliAuth {
			caCert, err := ioutil.ReadFile(s.caCert)
			if err != nil {
				return err
			}
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
			tlsConfig.ClientCAs = caCertPool
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		}
		keycert, err := tls.LoadX509KeyPair(s.svrCert, s.privKey)
		if err != nil {
			return err
		}
		tlsConfig.Certificates = append(tlsConfig.Certificates, keycert)

		server.ListenTCPTLS(s.bind, tlsConfig)
	}

	server.Boot()

	s.sls = server

	return nil
}

func (s *SyslogSvrInput) Read(ctx context.Context) (*service.Message, service.AckFunc, error) {

	select {
	case slsmsg := <-*s.slschan: // read a message from the channel
		msg := service.NewMessage(nil)
		msg.SetStructured(slsmsg)

		return msg, func(ctx context.Context, err error) error {
			return nil
		}, nil
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
}

func (s *SyslogSvrInput) Close(ctx context.Context) error {
	s.sls.Kill()
	return nil
}

//------------------------------------------------------------------------------

func main() {
	// SyslogServer Input Config Spec
	slsConfigSpec := service.NewConfigSpec().
		Summary("Creates a Syslog Server listener").
		Field(service.NewStringField("bind").
			Default("0.0.0.0:514").
			Description("The Listening IP address and port.")).
		Field(service.NewStringField("rfc").
			Default("Automatic").
			Description("The rfc format for the server: RFC3164, RFC5424, RFC6587 or Automatic")).
		Field(service.NewStringField("protocol").
			Default("UDP").
			Description("The server protocol - UDP or TCP")).
		Field(service.NewStringField("svrCert").
			Description("The signed cert for the TLS-enabled server service")).
		Field(service.NewStringField("privKey").
			Description("The private key for the TLS-enabled server service")).
		Field(service.NewStringField("caCert").
			Description("The CA Public certificate for client connections").Optional()).
		Field(service.NewBoolField("useTLS").
			Default(false).
			Description("Enforce TLS encryption")).
		Field(service.NewBoolField("cliAuth").
			Default(false).
			Description("Enforce Client certificate auth."))

	// Syslog Server Input constructor
	slsConstructor := func(conf *service.ParsedConfig, mgr *service.Resources) (service.Input, error) {
		bind, err := conf.FieldString("bind")
		if err != nil {
			return nil, err
		}
		rfc, err := conf.FieldString("rfc")
		if err != nil {
			return nil, err
		}
		protocol, err := conf.FieldString("protocol")
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
		useTLS, err := conf.FieldBool("useTLS")
		if err != nil {
			return nil, err
		}
		cliAuth, err := conf.FieldBool("cliAuth")
		if err != nil {
			return nil, err
		}
		var sls syslog.Server
		var slschan syslog.LogPartsChannel
		return service.AutoRetryNacks(&SyslogSvrInput{bind, rfc, protocol, svrCert, privKey, caCert, useTLS, cliAuth, &sls, &slschan}), nil
	}
	// Register Lumberjack Input
	err := service.RegisterInput("sysloginput", slsConfigSpec, slsConstructor)
	if err != nil {
		panic(err)
	}

	service.RunCLI(context.Background())
}
