# Benthos-Plugins
Shared plugins for feedback


**Lumberjack Server Input**

The Lumberjack protocol is used by the Elasticsearch beats agents, like Winlogbeat, to send batches of messages to Logstash, for example.

This plugin starts a server listener and reads message batches as an input to Benthos. This allows Benthos to replace Logstash and route and transform logs using all the Benthos goodness.

To test, configure the input with the following fields:
- bind: "[ip]:\<port\>"
- svrCert: "\<path to server cert file for service TLS\>"
- privKey: "\<path to server cert private key\>"
- caCert: "\<path to ca cert, if using client auth\>"
- cliAuth: false/true

To send Windows Log Events to the input listener, configure the Winlogbeat client's Logstash input section.

The plugin leverages the Elastic/go-lumber project.
For reference, there is an example server implementation at https://github.com/elastic/go-lumber/blob/main/cmd/tst-lj/main.go

  
**Syslog Server Input**

This plugin starts a syslog listener as an input for Benthos (TCP or UDP, and support TLS).

I have tested it with the stdout and Pulsar outputs. To quickly test the service with the stdout output, can use the following config file:

```
input:
 label: "syslog"
 sysloginput:
   bind: <IP_Addr>:<port>
   rfc: Automatic
   protocol: UDP
   svrCert: ""
   privKey: ""
   caCert: ""
   useTLS: false
   cliAuth: false
buffer:
 none: {}
pipeline:
 threads: -1
 processors: []
output:
 label: "stdout"
 stdout:
   codec: lines
 
shutdown_timeout: 10s
```
