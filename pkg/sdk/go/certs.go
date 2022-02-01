package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/mainflux/mainflux/pkg/errors"
)

const certsEndpoint = "certs"

// Cert represents certs data.
type Cert struct {
	CACert     string `json:"issuing_ca,omitempty"`
	ClientKey  string `json:"client_key,omitempty"`
	ClientCert string `json:"client_cert,omitempty"`
}

func (sdk mfSDK) IssueCert(thingID string, keyBits int, keyType, valid, token string) (Cert, error) {
	var c Cert
	r := certReq{
		ThingID: thingID,
		KeyBits: keyBits,
		KeyType: keyType,
		Valid:   valid,
	}
	d, err := json.Marshal(r)
	if err != nil {
		return Cert{}, err
	}

	url := fmt.Sprintf("%s/%s", sdk.certsURL, certsEndpoint)
	res, err := request(http.MethodPost, token, url, d)
	if err != nil {
		return Cert{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
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
		return errors.ErrAuthorization
	default:
		return ErrCertsRemove
	}
}

func (sdk mfSDK) RevokeCert(thingID, certID string, token string) error {
	panic("not implemented")
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
	ThingID    string `json:"thing_id"`
	KeyBits    int    `json:"key_bits"`
	KeyType    string `json:"key_type"`
	Encryption string `json:"encryption"`
	Valid      string `json:"valid"`
}
