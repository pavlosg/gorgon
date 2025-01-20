package jrpc

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"hash"
)

type authenticator struct {
	conn *BufferedStream
	mac  hash.Hash
}

var (
	errNilResult         = errors.New("jrpc: nil result")
	errUnexpectedRequest = errors.New("jrpc: unexpected request")
	errInvalidClientTag  = errors.New("jrpc: invalid client tag")
	errInvalidServerTag  = errors.New("jrpc: invalid server tag")
)

func newAuthenticator(conn *BufferedStream, key []byte) *authenticator {
	mac := hmac.New(sha256.New, key)
	mac.Reset()
	return &authenticator{conn, mac}
}

func (auth *authenticator) authClient() error {
	salt, err := randHex()
	if err != nil {
		return err
	}
	auth.sendRequest(authRequest{Method: "gorgon.auth0", Params: []string{salt}})
	_, err = auth.receiveResponse()
	if err != nil {
		return err
	}
	err = auth.sendRequest(authRequest{
		Method: "gorgon.auth1",
		Params: []string{auth.getTag()}})
	if err != nil {
		return err
	}
	expectedTag := auth.getTag()
	res, err := auth.receiveResponse()
	if err != nil {
		return err
	}
	if expectedTag != *res.Result {
		return errInvalidServerTag
	}
	return nil
}

func (auth *authenticator) authServer() error {
	req, err := auth.receiveRequest()
	if err != nil {
		return err
	}
	if req.Method != "gorgon.auth0" || len(req.Params) != 1 {
		return auth.sendError(errUnexpectedRequest)
	}
	salt, err := randHex()
	if err != nil {
		return auth.sendError(err)
	}
	err = auth.sendResult(salt)
	if err != nil {
		return err
	}
	expectedTag := auth.getTag()
	req, err = auth.receiveRequest()
	if err != nil {
		return err
	}
	if req.Method != "gorgon.auth1" || len(req.Params) != 1 {
		return auth.sendError(errUnexpectedRequest)
	}
	if expectedTag != req.Params[0] {
		return auth.sendError(errInvalidClientTag)
	}
	err = auth.sendResult(auth.getTag())
	if err != nil {
		return err
	}
	return nil
}

func randHex() (string, error) {
	var bytes [16]byte
	n, err := rand.Read(bytes[:])
	if err != nil {
		return "", err
	}
	if n < len(bytes) {
		panic("jrpc: crypto/rand.Read returned early")
	}
	return hex.EncodeToString(bytes[:]), nil
}

func (auth *authenticator) getTag() string {
	t := make([]byte, 0, auth.mac.Size())
	t = auth.mac.Sum(t)
	return hex.EncodeToString(t)
}

func (auth *authenticator) sendRequest(req authRequest) error {
	line, err := json.Marshal(req)
	if err != nil {
		return err
	}
	line = append(line, '\n')
	rem := line
	for len(rem) != 0 {
		n, err := auth.conn.Write(rem)
		if err != nil {
			return err
		}
		rem = rem[n:]
	}
	auth.mac.Write(line)
	return nil
}

func (auth *authenticator) sendResult(result string) error {
	return auth.sendResponce(authResponse{Result: &result})
}

func (auth *authenticator) sendError(err error) error {
	s := err.Error()
	auth.sendResponce(authResponse{Error: &s})
	return err
}

func (auth *authenticator) sendResponce(res authResponse) error {
	line, err := json.Marshal(res)
	if err != nil {
		return err
	}
	line = append(line, '\n')
	rem := line
	for len(rem) != 0 {
		n, err := auth.conn.Write(rem)
		if err != nil {
			return err
		}
		rem = rem[n:]
	}
	auth.mac.Write(line)
	return nil
}

func (auth *authenticator) receiveRequest() (req authRequest, err error) {
	line, err := auth.conn.readLine()
	if err != nil {
		return
	}
	auth.mac.Write(line)
	err = json.Unmarshal(line, &req)
	return
}

func (auth *authenticator) receiveResponse() (res authResponse, err error) {
	line, err := auth.conn.readLine()
	if err != nil {
		return
	}
	auth.mac.Write(line)
	err = json.Unmarshal(line, &res)
	if err != nil {
		return
	}
	if res.Error != nil {
		err = errors.New("jrpc: " + *res.Error)
	} else if res.Result == nil {
		err = errNilResult
	}
	return
}

type authRequest struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
}

type authResponse struct {
	Result *string `json:"result"`
	Error  *string `json:"error"`
}
