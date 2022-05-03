# Benthos-Plugins
Shared plugins for feedback


**Lumberjack Server Input**

The Lumberjack protocol is used by the Elasticsearch beats agents, like Winlogbeat, to send batches of messages to Logstash, for example.

This plugin is intended to start a server listener and read the message batches as an input to Benthos.

To test, configure the input, provide the following fields:

  input:
    label: "lumberjack"
    lumberjack:
      bind:   ":5044"
      svrCert:  ""
      privKey:  ""
      caCert: ""
      cliAuth: false
