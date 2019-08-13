package main

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
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	sdk "github.com/mainflux/mainflux/sdk/go"
)

type connection struct {
	ChannelID string
	ThingID   string
	ThingKey  string
	MTLSCert  string
	MTLSKey   string
}
type connections struct {
	Connection []connection
}

const (
	CRT_LOCATION  = "certs"
	KEY           = "default"
	Organization  = "Mainflux"
	OU            = "mainflux"
	EA            = "info@mainflux.com"
	CN            = "localhost"
	CRT_FILE_NAME = "thing"
	rsaBits       = 4096
	daysValid     = 730
)

func main() {
	var (
		host     = flag.String("host", "http://localhost", "Mainflux host address")
		username = flag.String("username", "mirkot@mainflux.com", "mainflux user")
		password = flag.String("password", "test1234", "mainflux user password")
		num      = flag.Int("num", 1, "number of created channels")
		ssl      = flag.Bool("ssl", true, "create thing certs")
		ca       = flag.String("ca", "ca.crt", "CA file for creating things certs")
		cakey    = flag.String("cakey", "ca.key", "CA private key file")
		prefix   = flag.String("prefix", "test", "name prefix for created channels and things")
	)

	flag.Parse()

	msgContentType := string(sdk.CTJSONSenML)
	sdkConf := sdk.Config{
		BaseURL:           *host,
		ReaderURL:         "http://localhost:8905",
		ReaderPrefix:      "",
		UsersPrefix:       "",
		ThingsPrefix:      "",
		HTTPAdapterPrefix: "http",
		MsgContentType:    sdk.ContentType(msgContentType),
		TLSVerification:   false,
	}

	s := sdk.NewSDK(sdkConf)

	user := sdk.User{
		Email:    *username,
		Password: *password,
	}

	token, err := s.CreateToken(user)
	if err != nil {
		log.Fatalf(err.Error())
		return
	}

	things := []sdk.Thing{}
	channels := []sdk.Channel{}
	connections := connections{Connection: []connection{}}
	var tlsCert tls.Certificate
	var caCert *x509.Certificate

	if *ssl {
		tlsCert, err = tls.LoadX509KeyPair(*ca, *cakey)
		if err != nil {
			log.Fatalf("Failed to load CA cert")
		}

		b, err := ioutil.ReadFile(*ca)
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

	for i := 0; i < *num; i++ {

		m, err := createThing(s, fmt.Sprintf("%s-thing-%d", *prefix, i), token)
		if err != nil {
			log.Println("Failed to create thing")
			return
		}

		ch, err := createChannel(s, fmt.Sprintf("%s-channel-%d", *prefix, i), token)

		if err := s.ConnectThing(m.ID, ch.ID, token); err != nil {
			log.Println("Failed to create thing")
			return
		}

		channels = append(channels, ch)
		m, err = s.Thing(m.ID, token)
		things = append(things, m)
		cert := ""
		key := ""
		if *ssl {
			var priv interface{}
			priv, err = rsa.GenerateKey(rand.Reader, rsaBits)

			notBefore := time.Now()
			notAfter := notBefore.Add(daysValid)

			serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
			serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
			if err != nil {
				log.Fatalf("failed to generate serial number: %s", err)
			}

			tmpl := x509.Certificate{
				SerialNumber: serialNumber,
				Subject: pkix.Name{
					Organization:       []string{Organization},
					CommonName:         m.Key,
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
				log.Fatalf("failed to write data to cert.pem: %s", err)
			}
			buffWriter.Flush()

			cert = bw.String()

			if err := pem.Encode(buffKeyOut, pemBlockForKey(priv)); err != nil {
				log.Fatalf("failed to write data to key.pem: %s", err)
			}

			buffKeyOut.Flush()
			key = keyOut.String()

		}
		connections.Connection = append(connections.Connection, connection{ch.ID, m.ID, m.Key, cert, key})

	}
	writeConnsToToml(connections)
}

func writeConnsToToml(c connections) {

	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(c); err != nil {
		log.Fatal(err)
	}

	fo, err := os.Create("channels.toml")
	if err != nil {
		panic(err)
	}

	if _, err := fo.Write(buf.Bytes()); err != nil {
		panic(err)
	}

}

func createThing(s sdk.SDK, name, token string) (sdk.Thing, error) {
	id, err := s.CreateThing(sdk.Thing{Name: name}, token)
	if err != nil {
		return sdk.Thing{}, err
	}

	t, err := s.Thing(id, token)
	if err != nil {
		return sdk.Thing{}, err
	}

	m := sdk.Thing{
		ID:   id,
		Name: name,
		Key:  t.Key,
	}
	return m, nil
}

func createChannel(s sdk.SDK, name, token string) (sdk.Channel, error) {
	id, err := s.CreateChannel(sdk.Channel{Name: name}, token)
	if err != nil {
		return sdk.Channel{}, nil
	}
	c := sdk.Channel{
		ID:   id,
		Name: name,
	}

	return c, nil
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
