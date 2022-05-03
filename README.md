# Benthos-Plugins
Shared plugins for feedback


**Lumberjack Server Input**

The Lumberjack protocol is used by the Elasticsearch beats agents, like Winlogbeat, to send batches of messages to Logstash, for example.

This plugin is intended to start a server listener and read the message batches as an input to Benthos.

To test, configure the input with the following fields:
- bind: "[ip]:\<port\>"
- svrCert: "\<path to server cert file for service TLS\>"
- privKey: "\<path to server cert private key\>"
- caCert: "\<path to ca cert, if using client auth\>"
- cliAuth: false/true

To send events to the input listener, configure the Winlogbeat client's Logstash input section.

The plugin leverages the Elastic/go-lumber project.
For reference, there is an example server implementation at https://github.com/elastic/go-lumber/blob/main/cmd/tst-lj/main.go
  
