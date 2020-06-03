package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// Cert represents certs data.
type Cert struct {
	CACert     string `json:"ca_cert,omitempty"`
	ClientKey  string `json:"client_key,omitempty"`
	ClientCert string `json:"client_cert,omitempty"`
}

func (sdk mfSDK) Cert(thingID, thingKey, token string) (Cert, error) {
	var c Cert
	r := certReq{
		ThingID:  thingID,
		ThingKey: thingKey,
	}
	d, err := json.Marshal(r)
	if err != nil {
		return Cert{}, err
	}
	res, err := request(http.MethodPost, token, sdk.certsURL, d)
	if err != nil {
		return Cert{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		return Cert{}, ErrCerts
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		println(err.Error())
		return Cert{}, err
	}
	if err := json.Unmarshal(body, &c); err != nil {
		return Cert{}, err
	}
	return c, nil
}

func (sdk mfSDK) RemoveCert(id, token string) error {
	res, err := request(http.MethodDelete, token, fmt.Sprintf("%s/%s", sdk.certsURL, id), nil)
	if res != nil {
		res.Body.Close()
	}
	if err != nil {
		return err
	}
	switch res.StatusCode {
	case http.StatusNoContent:
		return nil
	case http.StatusForbidden:
		return ErrUnauthorized
	default:
		return ErrCertsRemove
	}
}

func request(method, jwt, url string, data []byte) (*http.Response, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", jwt)
	c := &http.Client{}
	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

type certReq struct {
	ThingID  string `json:"thing_id,omitempty"`
	ThingKey string `json:"thing_key,omitempty"`
}
