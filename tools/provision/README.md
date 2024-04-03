# Magistrala Things and Channels Provisioning Tool

A simple utility to create a list of channels and things connected to these channels with possibility to create certificates for mTLS use case.

This tool is useful for testing, and it creates a TOML format output (on stdout, can be redirected into the file as needed)
that can be used by Magistrala MQTT benchmarking tool (`mqtt-bench`).

## Installation
```
cd tools/provision
make
```

### Usage
```
./provision --help
Tool for provisioning series of Magistrala channels and things and connecting them together.
Complete documentation is available at https://docs.magistrala.abstractmachines.fr

Usage:
  provision [flags]

Flags:
      --ca string         CA for creating and signing things certificate (default "ca.crt")
      --cakey string      ca.key for creating and signing things certificate (default "ca.key")
  -h, --help              help for provision
      --host string       address for magistrala instance (default "https://localhost")
      --num int           number of channels and things to create and connect (default 10)
  -p, --password string   magistrala users password
      --ssl               create certificates for mTLS access
  -u, --username string   magistrala user
      --prefix string     name prefix for things and channels
```

Example:
```
go run tools/provision/cmd/main.go -u test@magistrala.com -p test1234 --host https://142.93.118.47
```

If you want to create a list of channels with certificates:

```
go run tools/provision/cmd/main.go  --host http://localhost --num 10 -u test@magistrala.com -p test1234 --ssl true --ca docker/ssl/certs/ca.crt --cakey docker/ssl/certs/ca.key

```

>`ca.crt` and `ca.key` are used for creating things certificate and for HTTPS,
> if you are provisioning on remote server you will have to get these files to your local
> directory so that you can create certificates for things


Example of output:

```
# List of things that can be connected to MQTT broker
[[things]]
thing_id = "0eac601b-6d54-4767-b8b7-594aaf9990d3"
thing_key = "07713103-513f-43c7-b7fe-500c1af23d7d"
mtls_cert = """-----BEGIN CERTIFICATE-----
MIIEmTCCA4GgAwIBAgIRAO50qOfXsU+cHm/QY2NYu+0wDQYJKoZIhvcNAQELBQAw
VzESMBAGA1UEAwwJbG9jYWxob3N0MREwDwYDVQQKDAhNYWluZmx1eDEMMAoGA1UE
CwwDSW9UMSAwHgYJKoZIhvcNAQkBFhFpbmZvQG1haW5mbHV4LmNvbTAeFw0xOTEx
MTUxNzU2MzhaFw0yMDAyMjMxNzU2MzhaMFUxETAPBgNVBAoTCE1haW5mbHV4MREw
DwYDVQQLEwhtYWluZmx1eDEtMCsGA1UEAxMkMDc3MTMxMDMtNTEzZi00M2M3LWI3
ZmUtNTAwYzFhZjIzZDdkMIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEA
zsIYoovZJGJxfu7e4X3P3wnHDi9/wvRMhGW1EZEB5vNvfxvmmt4PhiE1c73mCypT
AUdui0j+hrCx8P90v12LEcJqty3yBnw+ge2/xCLNLKZh2/MjBQ7A7PMQpmOo31LR
hxFSthW41C296iwVYyvRa19y7g5mcUrzWvI2EVZbbGEDym1U/PI4aKhdQ3a7fF6B
GfvXYbGOa4/8VUIj8KHTRg2Z6/iLhxYgUnHd3xMCjihQkwLvB7/avVr9Ih9oLEe+
h7H9Pl5hMEpHP4BvHokUFhtbzqofuHNBKuEUf5r/cQ1oVAl6F77Fs5vZbQ59bLxw
etclDxW7nvOgIxEIUcJAkdd+nOxhpfbDM8QFsPXGSfb9vWUTaoQDIeWx9pPY5tsY
tbtW2HeKRGHO9jGFSzonY6sbTiaIzQ0F2PNPS1BoBIo2A95YNwt2ScfuRTs5ZK62
2+RNWbs+pDXJ5ZGcWDfjSxEYXy+jGUyvDExGCtryUu5Ufp7XuZ4O767iDzaj7dFG
rXSXfXrqwm8u2CMwucNzdVqikNG2gDToHDyIjLRd62m2pHk9gXbk3FGI+5x52pBs
+xdRaddMY8+DJ2R88PFoq3kqexxs2HJathCu6RfoP452zH9iU0gvPLR7fXuPoZ6Y
5NqE1CebZ6IiwwivD7kU1LxmhmQUY9DaHdHNYd66bd0CAwEAAaNiMGAwDgYDVR0P
AQH/BAQDAgeAMB0GA1UdJQQWMBQGCCsGAQUFBwMCBggrBgEFBQcDATAOBgNVHQ4E
BwQFAQIDBAYwHwYDVR0jBBgwFoAUbOMUfdahIzURpsN/dcUu8ek3PvIwDQYJKoZI
hvcNAQELBQADggEBAI+DdKYKKPVi4CPUbl+R81dq+Otd8L9i/RxM7G89XU0aGkSO
GSJzURKYbmLGgWdVWcdYMUfbpiE8vH1dLuDQdRywpDDjSMx7h0PwpYvk25HHKMSs
OIKpxvI1DyuNcwxrPuH863zw1Mo1hpGGin7yZc8VBf6nbR3RMNbQ2elMH1m7no4v
YM4HrTeR9n1bakIVw9OLnFpB03sT3keBdWsLDbAZ0yZfvxqdn6Hr7NRnab3vyrOz
GrYPJ51B/FGZC9n0ZR+SWzipen15vaG46SvoCv9HfDZ9cbSVR4eyPy/OIx+5CBVY
uGpJ+kN8jH5tuoxrmHZOsPMA+a6CZD2cKTaRu+Y=
-----END CERTIFICATE-----
"""
mtls_key = """-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----
"""

# List of channels that things can publish to
# each channel is connected to each thing from things list
# Things connected to channel 1f18afa1-29c4-4634-99d1-68dfa1b74e6a: 0eac601b-6d54-4767-b8b7-594aaf9990d3
[[channels]]
channel_id = "1f18afa1-29c4-4634-99d1-68dfa1b74e6a"

```
