package api

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"google.golang.org/grpc/codes"

	"github.com/asaskevich/govalidator"
	mux "github.com/dereulenspiegel/coap-mux"
	gocoap "github.com/dustin/go-coap"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/coap"
	"google.golang.org/grpc/status"
)

const (
	chanID    = "id"
	keyHeader = "key"
)

func authKey(opt interface{}) (string, error) {
	val, ok := opt.(string)
	if !ok {
		return "", errBadRequest
	}
	arr := strings.Split(val, "=")
	if len(arr) != 2 || strings.ToLower(arr[0]) != keyHeader {
		return "", errBadOption
	}
	return arr[1], nil
}

func authorize(msg *gocoap.Message, res *gocoap.Message, cid string) (publisher *mainflux.Identity, err error) {
	if !govalidator.IsUUID(cid) {
		res.Code = gocoap.NotFound
		return
	}

	// Device Key is passed as Uri-Query parameter, which option ID is 15 (0xf).
	key, err := authKey(msg.Option(gocoap.URIQuery))
	if err != nil {
		switch err {
		case errBadOption:
			res.Code = gocoap.BadOption
		case errBadRequest:
			res.Code = gocoap.BadRequest
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	publisher, err = auth.CanAccess(ctx, &mainflux.AccessReq{Token: key, ChanID: cid})

	if err != nil {
		e, ok := status.FromError(err)
		if ok {
			switch e.Code() {
			case codes.PermissionDenied:
				res.Code = gocoap.Forbidden
			default:
				res.Code = gocoap.ServiceUnavailable
			}
			return
		}
		res.Code = gocoap.InternalServerError
	}
	return
}

func serve(svc coap.Service, conn *net.UDPConn, data []byte, addr *net.UDPAddr, rh gocoap.Handler) {
	msg, err := gocoap.ParseMessage(data)
	if err != nil {
		return
	}
	res := &gocoap.Message{
		Type:      gocoap.NonConfirmable,
		Code:      gocoap.Content,
		MessageID: msg.MessageID,
		Token:     msg.Token,
		Payload:   []byte{},
	}

	switch msg.Type {
	case gocoap.Reset:
		if len(msg.Payload) != 0 {
			res.Code = gocoap.BadRequest
			break
		}
		cid := mux.Var(&msg, chanID)
		res.Type = gocoap.Acknowledgement
		publisher, err := authorize(&msg, res, cid)
		if err != nil {
			break
		}
		id := fmt.Sprintf("%s-%x", publisher, msg.Token)
		svc.RemoveTimeout(id)
		svc.Unsubscribe(id)
	case gocoap.Acknowledgement:
		cid := mux.Var(&msg, chanID)
		res.Type = gocoap.Acknowledgement
		publisher, err := authorize(&msg, res, cid)
		if err != nil {
			break
		}
		id := fmt.Sprintf("%s-%x", publisher, msg.Token)
		svc.RemoveTimeout(id)
	default:
		res = rh.ServeCOAP(conn, addr, &msg)
	}
	if res != nil && msg.IsConfirmable() {
		gocoap.Transmit(conn, addr, *res)
	}
}

// ListenAndServe binds to the given address and serve requests forever.
func ListenAndServe(svc coap.Service, csc mainflux.ClientsServiceClient, addr string) error {

	handler := makeHandler(addr, svc)
	auth = csc
	uaddr, err := net.ResolveUDPAddr(network, addr)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP(network, uaddr)
	if err != nil {
		return err
	}

	buf := make([]byte, maxPktLen)
	for {
		nr, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			if neterr, ok := err.(net.Error); ok && (neterr.Temporary() || neterr.Timeout()) {
				time.Sleep(5 * time.Millisecond)
				continue
			}
			return err
		}
		tmp := make([]byte, nr)
		copy(tmp, buf)
		go serve(svc, conn, tmp, addr, handler)
	}
}
