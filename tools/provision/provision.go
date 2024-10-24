// Copyright (c) Abstract Machines
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
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/0x6flab/namegenerator"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
)

const (
	defPass      = "12345678"
	defReaderURL = "http://localhost:9005"
)

var namesgenerator = namegenerator.NewGenerator()

// MgConn - structure describing Magistrala connection set.
type MgConn struct {
	ChannelID string
	ThingID   string
	ThingKey  string
	MTLSCert  string
	MTLSKey   string
}

// Config - provisioning configuration.
type Config struct {
	Host     string
	Username string
	Email    string
	Password string
	Num      int
	SSL      bool
	CA       string
	CAKey    string
	Prefix   string
}

// Provision - function that does actual provisiong.
func Provision(conf Config) error {
	const (
		rsaBits = 4096
		ttl     = "2400h"
	)

	msgContentType := string(sdk.CTJSONSenML)
	sdkConf := sdk.Config{
		ThingsURL:       conf.Host,
		UsersURL:        conf.Host,
		ReaderURL:       defReaderURL,
		HTTPAdapterURL:  fmt.Sprintf("%s/http", conf.Host),
		BootstrapURL:    conf.Host,
		CertsURL:        conf.Host,
		MsgContentType:  sdk.ContentType(msgContentType),
		TLSVerification: false,
	}

	s := sdk.NewSDK(sdkConf)

	user := sdk.User{
		Email: conf.Email,
		Credentials: sdk.Credentials{
			Username: conf.Username,
			Secret:   conf.Password,
		},
	}

	if user.Email == "" {
		user.Email = fmt.Sprintf("%s@email.com", namesgenerator.Generate())
		user.Credentials.Secret = defPass
	}

	// Create new user
	if _, err := s.CreateUser(user, ""); err != nil {
		return fmt.Errorf("unable to create new user: %s", err.Error())
	}

	var err error

	// Login user
	token, err := s.CreateToken(sdk.Login{Email: user.Email, Secret: user.Credentials.Secret})
	if err != nil {
		return fmt.Errorf("unable to login user: %s", err.Error())
	}

	// Create new domain
	dname := fmt.Sprintf("%s%s", conf.Prefix, namesgenerator.Generate())
	domain := sdk.Domain{
		Name:       dname,
		Alias:      strings.ToLower(dname),
		Permission: "admin",
	}

	domain, err = s.CreateDomain(domain, token.AccessToken)
	if err != nil {
		return fmt.Errorf("unable to create domain: %w", err)
	}
	// Login to domain
	token, err = s.CreateToken(sdk.Login{
		Email:    user.Email,
		Secret:   user.Credentials.Secret,
		DomainID: domain.ID,
	})
	if err != nil {
		return fmt.Errorf("unable to login user: %w", err)
	}

	var tlsCert tls.Certificate
	var caCert *x509.Certificate

	if conf.SSL {
		tlsCert, err = tls.LoadX509KeyPair(conf.CA, conf.CAKey)
		if err != nil {
			return fmt.Errorf("failed to load CA cert")
		}

		b, err := os.ReadFile(conf.CA)
		if err != nil {
			return fmt.Errorf("failed to load CA cert")
		}

		block, _ := pem.Decode(b)
		if block == nil {
			return fmt.Errorf("no PEM data found, failed to decode CA")
		}

		caCert, err = x509.ParseCertificate(block.Bytes)
		if err != nil {
			return fmt.Errorf("failed to decode certificate - %s", err.Error())
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

	things, err = s.CreateThings(things, domain.ID, token.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to create the things: %s", err.Error())
	}

	var chs []sdk.Channel
	for _, c := range channels {
		c, err = s.CreateChannel(c, domain.ID, token.AccessToken)
		if err != nil {
			return fmt.Errorf("failed to create the chennels: %s", err.Error())
		}
		chs = append(chs, c)
	}
	channels = chs

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
			priv, _ = rsa.GenerateKey(rand.Reader, rsaBits)

			notBefore := time.Now()
			validFor, err := time.ParseDuration(ttl)
			if err != nil {
				return fmt.Errorf("failed to set date %v", validFor)
			}
			notAfter := notBefore.Add(validFor)

			serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
			serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
			if err != nil {
				return fmt.Errorf("failed to generate serial number: %s", err)
			}

			tmpl := x509.Certificate{
				SerialNumber: serialNumber,
				Subject: pkix.Name{
					Organization:       []string{"Magistrala"},
					CommonName:         things[i].Credentials.Secret,
					OrganizationalUnit: []string{"magistrala"},
				},
				NotBefore: notBefore,
				NotAfter:  notAfter,

				KeyUsage:     x509.KeyUsageDigitalSignature,
				ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
				SubjectKeyId: []byte{1, 2, 3, 4, 6},
			}

			derBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, caCert, publicKey(priv), tlsCert.PrivateKey)
			if err != nil {
				return fmt.Errorf("failed to create certificate: %s", err)
			}

			var bw, keyOut bytes.Buffer
			buffWriter := bufio.NewWriter(&bw)
			buffKeyOut := bufio.NewWriter(&keyOut)

			if err := pem.Encode(buffWriter, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
				return fmt.Errorf("failed to write cert pem data: %s", err)
			}
			buffWriter.Flush()
			cert = bw.String()

			if err := pem.Encode(buffKeyOut, pemBlockForKey(priv)); err != nil {
				return fmt.Errorf("failed to write key pem data: %s", err)
			}
			buffKeyOut.Flush()
			key = keyOut.String()
		}

		// Print output
		fmt.Printf("[[things]]\nthing_id = \"%s\"\nthing_key = \"%s\"\n", things[i].ID, things[i].Credentials.Secret)
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

	for _, cID := range cIDs {
		for _, tID := range tIDs {
			conIDs := sdk.Connection{
				ThingID:   tID,
				ChannelID: cID,
			}
			if err := s.Connect(conIDs, domain.ID, token.AccessToken); err != nil {
				log.Fatalf("Failed to connect things %s to channels %s: %s", tID, cID, err)
			}
		}
	}

	return nil
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
