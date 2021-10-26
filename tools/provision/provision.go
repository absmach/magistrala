// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"bufio"
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/docker/docker/pkg/namesgenerator"
	sdk "github.com/mainflux/mainflux/pkg/sdk/go"
)

const (
	defPass      = "12345678"
	defReaderURL = "http://localhost:8905"
)

// MfConn - structure describing Mainflux connection set
type MfConn struct {
	ChannelID string
	ThingID   string
	ThingKey  string
	MTLSCert  string
	MTLSKey   string
}

// Config - provisioning configuration
type Config struct {
	Host     string
	Username string
	Password string
	Num      int
	SSL      bool
	CA       string
	CAKey    string
	Prefix   string
}

// Provision - function that does actual provisiong
func Provision(conf Config) {
	const (
		rsaBits   = 4096
		daysValid = "2400h"
	)

	msgContentType := string(sdk.CTJSONSenML)
	sdkConf := sdk.Config{
		ThingsURL:       conf.Host,
		ReaderURL:       defReaderURL,
		MsgContentType:  sdk.ContentType(msgContentType),
		TLSVerification: false,
	}

	s := sdk.NewSDK(sdkConf)

	user := sdk.User{
		Email:    conf.Username,
		Password: conf.Password,
	}

	if user.Email == "" {
		user.Email = fmt.Sprintf("%s@email.com", namesgenerator.GetRandomName(0))
		user.Password = defPass
	}

	// Create new user
	if _, err := s.CreateUser("", user); err != nil {
		log.Fatalf("Unable to create new user: %s", err.Error())
		return

	}

	// Login user
	token, err := s.CreateToken(user)
	if err != nil {
		log.Fatalf("Unable to login user: %s", err.Error())
		return
	}

	var tlsCert tls.Certificate
	var caCert *x509.Certificate

	if conf.SSL {
		tlsCert, err = tls.LoadX509KeyPair(conf.CA, conf.CAKey)
		if err != nil {
			log.Fatalf("Failed to load CA cert")
		}

		b, err := ioutil.ReadFile(conf.CA)
		if err != nil {
			log.Fatalf("Failed to load CA cert")
		}

		block, _ := pem.Decode(b)
		if block == nil {
			log.Fatalf("No PEM data found, failed to decode CA")
		}

		caCert, err = x509.ParseCertificate(block.Bytes)
		if err != nil {
			log.Fatalf("Failed to decode certificate - %s", err.Error())
		}

	}

	//  Create things and channels
	things := make([]sdk.Thing, conf.Num)
	channels := make([]sdk.Channel, conf.Num)
	cIDs := []string{}
	tIDs := []string{}

	fmt.Println("# List of things that can be connected to MQTT broker")

	for i := 0; i < conf.Num; i++ {
		things[i] = sdk.Thing{Name: fmt.Sprintf("%s-thing-%d", conf.Prefix, i)}
		channels[i] = sdk.Channel{Name: fmt.Sprintf("%s-channel-%d", conf.Prefix, i)}
	}

	things, err = s.CreateThings(things, token)
	if err != nil {
		log.Fatalf("Failed to create the things: %s", err.Error())
	}

	channels, err = s.CreateChannels(channels, token)
	if err != nil {
		log.Fatalf("Failed to create the chennels: %s", err.Error())
	}

	for _, t := range things {
		tIDs = append(tIDs, t.ID)
	}

	for _, c := range channels {
		cIDs = append(cIDs, c.ID)
	}

	for i := 0; i < conf.Num; i++ {
		cert := ""
		key := ""

		if conf.SSL {
			var priv interface{}
			priv, err = rsa.GenerateKey(rand.Reader, rsaBits)

			notBefore := time.Now()
			validFor, err := time.ParseDuration(daysValid)
			if err != nil {
				log.Fatalf("Failed to set date %v", validFor)
			}
			notAfter := notBefore.Add(validFor)

			serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
			serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
			if err != nil {
				log.Fatalf("Failed to generate serial number: %s", err)
			}

			tmpl := x509.Certificate{
				SerialNumber: serialNumber,
				Subject: pkix.Name{
					Organization:       []string{"Mainflux"},
					CommonName:         things[i].Key,
					OrganizationalUnit: []string{"mainflux"},
				},
				NotBefore: notBefore,
				NotAfter:  notAfter,

				KeyUsage:     x509.KeyUsageDigitalSignature,
				ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
				SubjectKeyId: []byte{1, 2, 3, 4, 6},
			}

			derBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, caCert, publicKey(priv), tlsCert.PrivateKey)
			if err != nil {
				log.Fatalf("Failed to create certificate: %s", err)
			}

			var bw, keyOut bytes.Buffer
			buffWriter := bufio.NewWriter(&bw)
			buffKeyOut := bufio.NewWriter(&keyOut)

			if err := pem.Encode(buffWriter, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
				log.Fatalf("Failed to write cert pem data: %s", err)
			}
			buffWriter.Flush()
			cert = bw.String()

			if err := pem.Encode(buffKeyOut, pemBlockForKey(priv)); err != nil {
				log.Fatalf("Failed to write key pem data: %s", err)
			}
			buffKeyOut.Flush()
			key = keyOut.String()
		}

		// Print output
		fmt.Printf("[[things]]\nthing_id = \"%s\"\nthing_key = \"%s\"\n", things[i].ID, things[i].Key)
		if conf.SSL {
			fmt.Printf("mtls_cert = \"\"\"%s\"\"\"\n", cert)
			fmt.Printf("mtls_key = \"\"\"%s\"\"\"\n", key)
		}
		fmt.Println("")
	}

	fmt.Printf("# List of channels that things can publish to\n" +
		"# each channel is connected to each thing from things list\n")
	for i := 0; i < conf.Num; i++ {
		fmt.Printf("[[channels]]\nchannel_id = \"%s\"\n\n", cIDs[i])
	}

	conIDs := sdk.ConnectionIDs{
		ChannelIDs: cIDs,
		ThingIDs:   tIDs,
	}
	if err := s.Connect(conIDs, token); err != nil {
		log.Fatalf("Failed to connect things %s to channels %s: %s", conIDs.ThingIDs, conIDs.ChannelIDs, err)
	}
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func pemBlockForKey(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to marshal ECDSA private key: %v", err)
			os.Exit(2)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	default:
		return nil
	}
}
