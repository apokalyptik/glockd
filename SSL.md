# Enabling

Enable TLS encryption and client authentication with the `-ssl` flag.
Doing so requires that you provide values for 3 other flags:

* `-ssl-ca` to specify the certificate authority to validate the client certificates against
* `-ssl-cert` to specify the PEM certificate to present to clients
* `-ssl-key` to specify the key file relating to the certificate being presented to clients

When you have done this the TCP connections on the -port (currently only this, not the -ws connections) will have to be TLS connections.

# Testing

## Certificate Authority and Certificates

If you don't have a CA already set up (I know I didn't) and want to test the following bash script will both make a 
`certificate-authority.pem` file to use with `-ssl-ca` but also a signed .key file for `-ssl-key` and .pem file for
`-ssl-cert`. You can run it with various host names and IP addresses to test different configurations and also for
generating client certificates.

```bash
#!/bin/bash

set -xe

IP="$1"
HOST="$2"
FILE=$(echo "$IP  $HOST" | sed -r -e 's/[^0-9a-zA-Z]/-/g')

if [ "$IP" = "" ]; then
	echo "$0 \'ip[ ip[ ...]]\' \'host[ host [...]]\']"
	exit 1
fi

newIP=""
n=0
for i in $IP; do
	let n=$n+1
	if [ "$newIP" = "" ]; then
		newIP="IP.$n:$i"
	else
		newIP="$newIP,IP.$n:$i"
	fi
done
IP="$newIP"

newHost=""
n=0
for i in $HOST; do
	let n=$n+1
	if [ "$newHost" = "" ]; then
		newHost="DNS.$n:$i"
	else
		newHost="$newHost,DNS.$n:$i"
	fi
done
HOST="$newHost"

IP="$newIP"
if [ ! -f certificate-authority.key ] || [ ! -f certificate-authority.pem ]; then
openssl req \
    -newkey rsa:4096 \
    -nodes \
    -days 365000 \
    -x509 \
    -keyout certificate-authority.key \
    -out certificate-authority.pem \
    -subj "/CN=*"
fi

openssl req \
    -newkey rsa:4096 \
    -nodes \
    -keyout "$FILE.key" \
    -out "$FILE.csr" \
    -subj "/C=US/ST=MyST/L=MyL/O=MyO/OU=MyOU/CN=*" # CN= probably needs to be better?

openssl x509 \
    -req \
    -days 36500 \
    -sha256 \
    -in "$FILE.csr" \
    -CA certificate-authority.pem \
    -CAkey certificate-authority.key \
    -CAcreateserial \
    -out "$FILE.pem" \
    -extfile <(echo "subjectAltName=$IP,$HOST")
```

## Simple Go client test script

```go
package main

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
)

func main() {
	ca := x509.NewCertPool()
	rootPEM, err := ioutil.ReadFile("path/to/certificate-authority.pem")
	if err != nil {
		panic("failed to parse certificate-authority.pem: " + err.Error())
	}
	ok := ca.AppendCertsFromPEM(rootPEM)
	if !ok {
		panic("failed to parse root certificate")
	}
	cert, err := tls.LoadX509KeyPair("path/to/cert.pem", "path/to/key.key")
	if err != nil {
		panic("failed to parse client-1.*: " + err.Error())
	}
	log.Printf("%#v", cert)
	conn, err := tls.Dial("tcp", "127.0.0.1:9999", &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            ca,
		InsecureSkipVerify: false,
	})
	if err != nil {
		panic("failed to connect: " + err.Error())
	}
	conn.Close()
}
```

## Simple PHP client test script

```php
#!/usr/bin/env php
<?php
$opts = array(
	'ssl' => array(
		'verify_peer' => true,
		'verify_peer_name' => true,
		'cafile' => 'path/to/certificate-authority.pem',
		'local_cert' => 'path/to/cert.pem',
		'local_pk' => 'path/to/key.key',
	),
);
$context = stream_context_create($opts);
$fp = stream_socket_client("ssl://127.0.0.1:9999", $errno, $errst, 30, STREAM_CLIENT_CONNECT, $context);
var_dump( $fp );
var_dump( $errno );
var_dump( $errst );
fclose( $fp );
```
