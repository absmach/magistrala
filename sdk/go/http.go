//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package sdk

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/mainflux/mainflux"
)

const (
	contentTypeJSON      = "application/json"
	contentTypeSenMLJSON = "application/senml+json"
	contentTypeBinary    = "application/octet-stream"

	defCertsPath = "/src/github.com/mainflux/mainflux/docker/ssl/certs/"

	envCertFile = "MF_CERT_FILE"
	envKeyFile  = "MF_KEY_FILE"
	envCaFile   = "MF_CA_FILE"
)

var (
	defCertFile = fmt.Sprintf("%s%s%s", os.Getenv("GOPATH"), defCertsPath, "mainflux-server.crt")
	defKeyFile  = fmt.Sprintf("%s%s%s", os.Getenv("GOPATH"), defCertsPath, "mainflux-server.key")
	defCaFile   = fmt.Sprintf("%s%s%s", os.Getenv("GOPATH"), defCertsPath, "ca.crt")

	limit  = 10
	offset = 0
)

// setCerts - set TLS certs
// Certs are provided via MF_CERT_FILE, MF_KEY_FILE and MF_CA_FILE env vars
func setCerts() *http.Client {
	// Set certificates paths
	certFile := mainflux.Env(envCertFile, defCertFile)
	keyFile := mainflux.Env(envKeyFile, defKeyFile)
	caFile := mainflux.Env(envCaFile, defCaFile)

	// Load client cert
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatal(err)
	}

	// Load CA cert
	caCert, err := ioutil.ReadFile(caFile)
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Setup HTTPS client
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}
	tlsConfig.BuildNameToCertificate()
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	return &http.Client{Transport: transport}
}
