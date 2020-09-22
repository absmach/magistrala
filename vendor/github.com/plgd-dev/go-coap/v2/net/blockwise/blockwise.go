package blockwise

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	kitSync "github.com/plgd-dev/kit/sync"

	"github.com/dsnet/golib/memfile"
	"github.com/patrickmn/go-cache"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
)

// Block Opion value is represented: https://tools.ietf.org/html/rfc7959#section-2.2
//  0
//  0 1 2 3 4 5 6 7
// +-+-+-+-+-+-+-+-+
// |  NUM  |M| SZX |
// +-+-+-+-+-+-+-+-+
//  0                   1
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |          NUM          |M| SZX |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//  0                   1                   2
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                   NUM                 |M| SZX |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

const (
	// max block size is 3bytes: https://tools.ietf.org/html/rfc7959#section-2.1
	maxBlockValue = 0xffffff
	// maxBlockNumber is 20bits (NUM)
	maxBlockNumber = 0xffff7
	// moreBlocksFollowingMask is represented by one bit (M)
	moreBlocksFollowingMask = 0x8
	// szxMask last 3bits represents SZX (SZX)
	szxMask = 0x7
)

// SZX enum representation for the size of the block: https://tools.ietf.org/html/rfc7959#section-2.2
type SZX uint8

const (
	//SZX16 block of size 16bytes
	SZX16 SZX = 0
	//SZX32 block of size 32bytes
	SZX32 SZX = 1
	//SZX64 block of size 64bytes
	SZX64 SZX = 2
	//SZX128 block of size 128bytes
	SZX128 SZX = 3
	//SZX256 block of size 256bytes
	SZX256 SZX = 4
	//SZX512 block of size 512bytes
	SZX512 SZX = 5
	//SZX1024 block of size 1024bytes
	SZX1024 SZX = 6
	//SZXBERT block of size n*1024bytes
	SZXBERT SZX = 7
)

var szxToSize = map[SZX]int64{
	SZX16:   16,
	SZX32:   32,
	SZX64:   64,
	SZX128:  128,
	SZX256:  256,
	SZX512:  512,
	SZX1024: 1024,
	SZXBERT: 1024,
}

// Size number of bytes.
func (s SZX) Size() int64 {
	val, ok := szxToSize[s]
	if ok {
		return val
	}
	return -1
}

// ResponseWriter defines response interface for blockwise transfer.
type ResponseWriter interface {
	Message() Message
	SetMessage(Message)
}

// Message defines message interface for blockwise transfer.
type Message interface {
	// getters
	Context() context.Context
	Code() codes.Code
	Token() message.Token
	GetOptionUint32(id message.OptionID) (uint32, error)
	GetOptionBytes(id message.OptionID) ([]byte, error)
	Options() message.Options
	Body() io.ReadSeeker
	BodySize() (int64, error)
	Sequence() uint64
	// setters
	SetCode(codes.Code)
	SetToken(message.Token)
	SetOptionUint32(id message.OptionID, value uint32)
	Remove(id message.OptionID)
	ResetOptionsTo(message.Options)
	SetBody(r io.ReadSeeker)
	SetSequence(uint64)
	String() string
}

// EncodeBlockOption encodes block values to coap option.
func EncodeBlockOption(szx SZX, blockNumber int64, moreBlocksFollowing bool) (uint32, error) {
	if szx > SZXBERT {
		return 0, ErrInvalidSZX
	}
	if blockNumber < 0 {
		return 0, ErrBlockNumberExceedLimit
	}
	if blockNumber > maxBlockNumber {
		return 0, ErrBlockNumberExceedLimit
	}
	blockVal := uint32(blockNumber << 4)
	m := uint32(0)
	if moreBlocksFollowing {
		m = 1
	}
	blockVal += m << 3
	blockVal += uint32(szx)
	return blockVal, nil
}

// DecodeBlockOption decodes coap block option to block values.
func DecodeBlockOption(blockVal uint32) (szx SZX, blockNumber int64, moreBlocksFollowing bool, err error) {
	if blockVal > maxBlockValue {
		err = ErrBlockInvalidSize
		return
	}

	szx = SZX(blockVal & szxMask)                  //masking for the SZX
	if (blockVal & moreBlocksFollowingMask) != 0 { //masking for the "M"
		moreBlocksFollowing = true
	}
	blockNumber = int64(blockVal) >> 4 //shifting out the SZX and M vals. leaving the block number behind
	if blockNumber > maxBlockNumber {
		err = ErrBlockNumberExceedLimit
	}
	return
}

type BlockWise struct {
	acquireMessage              func(ctx context.Context) Message
	releaseMessage              func(Message)
	receivingMessagesCache      *cache.Cache
	sendingMessagesCache        *cache.Cache
	errors                      func(error)
	autoCleanUpResponseCache    bool
	getSendedRequestFromOutside func(token message.Token) (Message, bool)

	bwSendedRequest *kitSync.Map
}

type messageGuard struct {
	sync.Mutex
	request Message
}

func newRequestGuard(request Message) *messageGuard {
	return &messageGuard{
		request: request,
	}
}

// NewBlockWise provides blockwise.
// getSendedRequestFromOutside must returns a copy of request which will be released by function releaseMessage after use.
func NewBlockWise(
	acquireMessage func(ctx context.Context) Message,
	releaseMessage func(Message),
	expiration time.Duration,
	errors func(error),
	autoCleanUpResponseCache bool,
	getSendedRequestFromOutside func(token message.Token) (Message, bool),
) *BlockWise {
	receivingMessagesCache := cache.New(expiration, expiration)
	bwSendedRequest := kitSync.NewMap()
	receivingMessagesCache.OnEvicted(func(tokenstr string, _ interface{}) {
		bwSendedRequest.Delete(tokenstr)
	})
	if getSendedRequestFromOutside == nil {
		getSendedRequestFromOutside = func(token message.Token) (Message, bool) { return nil, false }
	}
	return &BlockWise{
		acquireMessage:              acquireMessage,
		releaseMessage:              releaseMessage,
		receivingMessagesCache:      receivingMessagesCache,
		sendingMessagesCache:        cache.New(expiration, expiration),
		errors:                      errors,
		autoCleanUpResponseCache:    autoCleanUpResponseCache,
		getSendedRequestFromOutside: getSendedRequestFromOutside,
		bwSendedRequest:             bwSendedRequest,
	}
}

func bufferSize(szx SZX, maxMessageSize int) int64 {
	if szx < SZXBERT {
		return szx.Size()
	}
	return (int64(maxMessageSize) / szx.Size()) * szx.Size()
}

func (b *BlockWise) newSendRequestMessage(r Message) Message {
	req := b.acquireMessage(r.Context())
	req.SetCode(r.Code())
	req.SetToken(r.Token())
	req.ResetOptionsTo(r.Options())
	return req
}

// Do sends an coap message and returns an coap response via blockwise transfer.
func (b *BlockWise) Do(r Message, maxSzx SZX, maxMessageSize int, do func(req Message) (Message, error)) (Message, error) {
	if maxSzx > SZXBERT {
		return nil, fmt.Errorf("invalid szx")
	}
	if len(r.Token()) == 0 {
		return nil, fmt.Errorf("invalid token")
	}

	req := b.newSendRequestMessage(r)
	defer b.releaseMessage(req)

	tokenStr := r.Token().String()
	b.bwSendedRequest.Store(tokenStr, req)
	defer b.bwSendedRequest.Delete(tokenStr)
	if r.Body() == nil {
		return do(r)
	}
	payloadSize, err := r.BodySize()
	if err != nil {
		return nil, fmt.Errorf("cannot get size of payload: %w", err)
	}
	if payloadSize <= int64(maxSzx.Size()) {
		return do(r)
	}

	switch r.Code() {
	case codes.POST, codes.PUT:
		break
	default:
		return nil, fmt.Errorf("unsupported command(%v)", r.Code())
	}
	req.SetOptionUint32(message.Size1, uint32(payloadSize))

	num := int64(0)
	buf := make([]byte, 1024)
	szx := maxSzx
	for {
		newBufLen := bufferSize(szx, maxMessageSize)
		if int64(cap(buf)) < newBufLen {
			buf = make([]byte, newBufLen)
		}
		buf = buf[:newBufLen]

		off := int64(num * szx.Size())
		newOff, err := r.Body().Seek(off, io.SeekStart)
		if err != nil {
			return nil, fmt.Errorf("cannot seek in payload: %w", err)
		}
		readed, err := io.ReadFull(r.Body(), buf)
		if err == io.ErrUnexpectedEOF {
			if newOff+int64(readed) == payloadSize {
				err = nil
			}
		}
		if err != nil {
			return nil, fmt.Errorf("cannot read payload: %w", err)
		}
		buf = buf[:readed]
		req.SetBody(bytes.NewReader(buf))
		more := true
		if newOff+int64(readed) == payloadSize {
			more = false
		}
		block, err := EncodeBlockOption(szx, num, more)
		if err != nil {
			return nil, fmt.Errorf("cannot encode block option(%v, %v, %v) to bw request: %w", szx, num, more, err)
		}

		req.SetOptionUint32(message.Block1, block)
		resp, err := do(req)
		if err != nil {
			return nil, fmt.Errorf("cannot do bw request: %w", err)
		}
		block, err = resp.GetOptionUint32(message.Block1)
		if err != nil {
			return resp, nil
		}
		switch resp.Code() {
		case codes.Continue:
		case codes.Created, codes.Changed:
			if !more {
				return resp, nil
			}
		default:
			return resp, nil
		}

		var newSzx SZX
		var newNum int64
		newSzx, newNum, _, err = DecodeBlockOption(block)
		if err != nil {
			return resp, fmt.Errorf("cannot decode block option of bw response: %w", err)
		}
		if num != newNum {
			return resp, fmt.Errorf("unexpected of acknowleged seqencenumber(%v != %v)", num, newNum)
		}

		num = num + newSzx.Size()/szx.Size()
		szx = newSzx
	}
}

type writeMessageResponse struct {
	request        Message
	releaseMessage func(Message)
}

func NewWriteRequestResponse(request Message, acquireMessage func(context.Context) Message, releaseMessage func(Message)) *writeMessageResponse {
	req := acquireMessage(request.Context())
	req.SetCode(request.Code())
	req.SetToken(request.Token())
	req.ResetOptionsTo(request.Options())
	req.SetBody(request.Body())
	return &writeMessageResponse{
		request:        req,
		releaseMessage: releaseMessage,
	}
}

func (w *writeMessageResponse) SetMessage(r Message) {
	w.releaseMessage(w.request)
	w.request = r
}

func (w *writeMessageResponse) Message() Message {
	return w.request
}

// WriteMessage sends an coap message via blockwise transfer.
func (b *BlockWise) WriteMessage(request Message, maxSZX SZX, maxMessageSize int, writeMessage func(r Message) error) error {
	req := b.newSendRequestMessage(request)
	tokenStr := req.Token().String()
	b.bwSendedRequest.Store(tokenStr, req)
	startSendingMessageBlock, err := EncodeBlockOption(maxSZX, 0, true)
	if err != nil {
		return fmt.Errorf("cannot encode start sending message block option(%v,%v,%v): %w", maxSZX, 0, true, err)
	}

	w := NewWriteRequestResponse(request, b.acquireMessage, b.releaseMessage)
	err = b.startSendingMessage(w, maxSZX, maxMessageSize, startSendingMessageBlock)
	if err != nil {
		return fmt.Errorf("cannot start writing request: %w", err)
	}
	return writeMessage(w.Message())
}

func fitSZX(r Message, blockType message.OptionID, maxSZX SZX) SZX {
	block, err := r.GetOptionUint32(blockType)
	if err == nil {
		szx, _, _, err := DecodeBlockOption(block)
		if err != nil {
			if maxSZX > szx {
				return szx
			}
		}
	}
	return maxSZX
}

func (b *BlockWise) handleSendingMessage(w ResponseWriter, sendingMessage Message, maxSZX SZX, maxMessageSize int, token []byte, block uint32) (bool, error) {
	blockType := message.Block2
	sizeType := message.Size2
	switch sendingMessage.Code() {
	case codes.POST, codes.PUT:
		blockType = message.Block1
		sizeType = message.Size1
	}

	szx, num, _, err := DecodeBlockOption(block)
	if err != nil {
		return false, fmt.Errorf("cannot decode %v option: %w", blockType, err)
	}
	off := int64(num * szx.Size())
	if szx > maxSZX {
		szx = maxSZX
	}
	sendMessage := b.acquireMessage(sendingMessage.Context())
	sendMessage.SetCode(sendingMessage.Code())
	sendMessage.ResetOptionsTo(sendingMessage.Options())
	sendMessage.SetToken(token)
	payloadSize, err := sendingMessage.BodySize()
	if err != nil {
		return false, fmt.Errorf("cannot get size of payload: %w", err)
	}
	offSeek, err := sendingMessage.Body().Seek(off, io.SeekStart)
	if err != nil {
		return false, fmt.Errorf("cannot seek in response: %w", err)
	}
	if off != offSeek {
		return false, fmt.Errorf("cannot seek to requested offset(%v != %v)", off, offSeek)
	}
	buf := make([]byte, 1024)
	newBufLen := bufferSize(szx, maxMessageSize)
	if int64(len(buf)) < newBufLen {
		buf = make([]byte, newBufLen)
	}
	buf = buf[:newBufLen]

	readed, err := io.ReadFull(sendingMessage.Body(), buf)
	if err == io.ErrUnexpectedEOF {
		if offSeek+int64(readed) == payloadSize {
			err = nil
		}
	}

	buf = buf[:readed]
	sendMessage.SetBody(bytes.NewReader(buf))
	more := true
	if offSeek+int64(readed) == payloadSize {
		more = false
	}
	sendMessage.SetOptionUint32(sizeType, uint32(payloadSize))
	num = (offSeek+int64(readed))/szx.Size() - (int64(readed) / szx.Size())
	block, err = EncodeBlockOption(szx, num, more)
	if err != nil {
		return false, fmt.Errorf("cannot encode block option(%v,%v,%v): %w", szx, num, more, err)
	}
	sendMessage.SetOptionUint32(blockType, block)
	w.SetMessage(sendMessage)
	return more, nil
}

// RemoveFromResponseCache removes response from cache. It need's tu be used for udp coap.
func (b *BlockWise) RemoveFromResponseCache(token message.Token) {
	if len(token) == 0 {
		return
	}
	b.sendingMessagesCache.Delete(token.String())
}

func (b *BlockWise) sendEntityIncomplete(w ResponseWriter, token message.Token) {
	sendMessage := b.acquireMessage(w.Message().Context())
	sendMessage.SetCode(codes.RequestEntityIncomplete)
	sendMessage.SetToken(token)
	w.SetMessage(sendMessage)
}

// Handle middleware which constructs COAP request from blockwise transfer and send COAP response via blockwise.
func (b *BlockWise) Handle(w ResponseWriter, r Message, maxSZX SZX, maxMessageSize int, next func(w ResponseWriter, r Message)) {
	if maxSZX > SZXBERT {
		panic("invalid maxSZX")
	}
	token := r.Token()

	if len(token) == 0 {
		err := b.handleReceivedMessage(w, r, maxSZX, maxMessageSize, next)
		if err != nil {
			b.sendEntityIncomplete(w, token)
			b.errors(fmt.Errorf("handleReceivedMessage(%v): %w", r, err))
		}
		return
	}
	tokenStr := token.String()
	v, ok := b.sendingMessagesCache.Get(tokenStr)

	if !ok {
		err := b.handleReceivedMessage(w, r, maxSZX, maxMessageSize, next)
		if err != nil {
			b.sendEntityIncomplete(w, token)
			b.errors(fmt.Errorf("handleReceivedMessage(%v): %w", r, err))
		}
		return
	}
	more, err := b.continueSendingMessage(w, r, maxSZX, maxMessageSize, v.(*messageGuard))
	if err != nil {
		b.sendingMessagesCache.Delete(tokenStr)
		b.errors(fmt.Errorf("continueSendingMessage(%v): %w", r, err))
		return
	}
	if b.autoCleanUpResponseCache && more == false {
		b.RemoveFromResponseCache(token)
	}
}

func (b *BlockWise) handleReceivedMessage(w ResponseWriter, r Message, maxSZX SZX, maxMessageSize int, next func(w ResponseWriter, r Message)) error {
	startSendingMessageBlock, err := EncodeBlockOption(maxSZX, 0, true)
	if err != nil {
		return fmt.Errorf("cannot encode start sending message block option(%v,%v,%v): %w", maxSZX, 0, true, err)
	}
	switch r.Code() {
	case codes.Empty:
		next(w, r)
		return nil
	case codes.CSM, codes.Ping, codes.Pong, codes.Release, codes.Abort, codes.Continue:
		next(w, r)
		return nil
	case codes.GET, codes.DELETE:
		maxSZX = fitSZX(r, message.Block2, maxSZX)
		block, err := r.GetOptionUint32(message.Block2)
		if err == nil {
			r.Remove(message.Block2)
		}
		next(w, r)
		if w.Message().Code() == codes.Content && err == nil {
			startSendingMessageBlock = block
		}
	case codes.POST, codes.PUT:
		maxSZX = fitSZX(r, message.Block1, maxSZX)
		err := b.processReceivedMessage(w, r, maxSZX, next, message.Block1, message.Size1)
		if err != nil {
			return err
		}
	default:
		maxSZX = fitSZX(r, message.Block2, maxSZX)
		err = b.processReceivedMessage(w, r, maxSZX, next, message.Block2, message.Size2)
		if err != nil {
			return err
		}

	}
	return b.startSendingMessage(w, maxSZX, maxMessageSize, startSendingMessageBlock)
}

func (b *BlockWise) continueSendingMessage(w ResponseWriter, r Message, maxSZX SZX, maxMessageSize int, messageGuard *messageGuard) (bool, error) {
	messageGuard.Lock()
	defer messageGuard.Unlock()
	resp := messageGuard.request
	blockType := message.Block2
	switch resp.Code() {
	case codes.POST, codes.PUT:
		blockType = message.Block1
	}

	block, err := r.GetOptionUint32(blockType)
	if err != nil {
		return false, fmt.Errorf("cannot get %v option: %w", blockType, err)
	}
	if blockType == message.Block1 {
		// num just acknowlege position we need to set block to next block.
		szx, num, more, err := DecodeBlockOption(block)
		if err != nil {
			return false, fmt.Errorf("cannot decode %v(%v) option: %w", blockType, block, err)
		}
		off, err := resp.Body().Seek(0, io.SeekCurrent)
		if err != nil {
			return false, fmt.Errorf("cannot get current position of seek: %w", err)
		}
		num = off / szx.Size()
		block, err = EncodeBlockOption(szx, num, more)
		if err != nil {
			return false, fmt.Errorf("cannot encode %v(%v, %v, %v) option: %w", blockType, szx, num, more, err)
		}
	}
	more, err := b.handleSendingMessage(w, resp, maxSZX, maxMessageSize, r.Token(), block)

	if err != nil {
		return false, fmt.Errorf("handleSendingMessage: %w", err)
	}
	return more, err
}

func isObserveResponse(msg Message) bool {
	_, err := msg.GetOptionUint32(message.Observe)
	if err != nil {
		return false
	}
	if msg.Code() == codes.Content {
		return true
	}
	return false
}

func (b *BlockWise) startSendingMessage(w ResponseWriter, maxSZX SZX, maxMessageSize int, block uint32) error {
	payloadSize, err := w.Message().BodySize()
	if err != nil {
		return fmt.Errorf("cannot get size of payload: %w", err)
	}

	if payloadSize < int64(maxSZX.Size()) {
		return nil
	}
	sendingMessage := b.acquireMessage(w.Message().Context())
	sendingMessage.ResetOptionsTo(w.Message().Options())
	sendingMessage.SetBody(w.Message().Body())
	sendingMessage.SetCode(w.Message().Code())
	sendingMessage.SetToken(w.Message().Token())

	_, err = b.handleSendingMessage(w, sendingMessage, maxSZX, maxMessageSize, sendingMessage.Token(), block)
	if err != nil {
		return fmt.Errorf("handleSendingMessage: %w", err)
	}
	if isObserveResponse(w.Message()) {
		// https://tools.ietf.org/html/rfc7959#section-2.6 - we don't need store it because client will be get values via GET.
		return nil
	}
	err = b.sendingMessagesCache.Add(sendingMessage.Token().String(), newRequestGuard(sendingMessage), cache.DefaultExpiration)
	if err != nil {
		return fmt.Errorf("cannot add to response cachce: %w", err)
	}
	return nil
}

func (b *BlockWise) getSendedRequest(token message.Token) Message {
	v, ok := b.bwSendedRequest.LoadWithFunc(token.String(), func(v interface{}) interface{} {
		r := v.(Message)
		return b.newSendRequestMessage(r)
	})
	if ok {
		return v.(Message)
	}
	globalRequest, ok := b.getSendedRequestFromOutside(token)
	if ok {
		return globalRequest
	}
	return nil
}

func (b *BlockWise) processReceivedMessage(w ResponseWriter, r Message, maxSzx SZX, next func(w ResponseWriter, r Message), blockType message.OptionID, sizeType message.OptionID) error {
	token := r.Token()
	if len(token) == 0 {
		next(w, r)
		return nil
	}
	if r.Code() == codes.GET || r.Code() == codes.DELETE {
		next(w, r)
		return nil
	}

	block, err := r.GetOptionUint32(blockType)
	if err != nil {

		next(w, r)
		return nil
	}
	szx, num, more, err := DecodeBlockOption(block)
	if err != nil {
		return fmt.Errorf("cannot decode block option: %w", err)
	}
	sendedRequest := b.getSendedRequest(token)
	if sendedRequest != nil {
		defer b.releaseMessage(sendedRequest)
	}
	if blockType == message.Block2 && sendedRequest == nil {
		return fmt.Errorf("cannot request body without paired request")
	}
	if isObserveResponse(r) {
		// https://tools.ietf.org/html/rfc7959#section-2.6 - performs GET with new token.
		if sendedRequest == nil {
			return fmt.Errorf("observation is not registered")
		}
		token, err = message.GetToken()
		if err != nil {
			return fmt.Errorf("cannot get token for create GET request: %w", err)
		}
		bwSendedRequest := b.acquireMessage(sendedRequest.Context())
		bwSendedRequest.SetCode(sendedRequest.Code())
		bwSendedRequest.SetToken(token)
		bwSendedRequest.ResetOptionsTo(sendedRequest.Options())
		b.bwSendedRequest.Store(token.String(), bwSendedRequest)
	}

	tokenStr := token.String()
	cachedReceivedMessageGuard, ok := b.receivingMessagesCache.Get(tokenStr)
	if !ok {
		if szx > maxSzx {
			szx = maxSzx
		}
		// first request must have 0
		if num != 0 {
			return fmt.Errorf("token %v, invalid %v(%v), expected 0", []byte(token), blockType, num)
		}
		// if there is no more then just forward req to next handler
		if more == false {
			next(w, r)
			return nil
		}
		cachedReceivedMessage := b.acquireMessage(r.Context())
		cachedReceivedMessage.ResetOptionsTo(r.Options())
		cachedReceivedMessage.SetToken(r.Token())
		cachedReceivedMessage.SetSequence(r.Sequence())
		cachedReceivedMessageGuard = newRequestGuard(cachedReceivedMessage)
		err := b.receivingMessagesCache.Add(tokenStr, cachedReceivedMessageGuard, cache.DefaultExpiration)
		// request was already stored in cache, silently
		if err != nil {
			return fmt.Errorf("request was already stored in cache")
		}
		cachedReceivedMessage.SetBody(memfile.New(make([]byte, 0, 1024)))
	}
	messageGuard := cachedReceivedMessageGuard.(*messageGuard)
	defer func(err *error) {
		if *err != nil {
			b.receivingMessagesCache.Delete(tokenStr)
		}
	}(&err)
	messageGuard.Lock()
	defer messageGuard.Unlock()
	cachedReceivedMessage := messageGuard.request
	rETAG, errETAG := r.GetOptionBytes(message.ETag)
	cachedReceivedMessageETAG, errCachedReceivedMessageETAG := cachedReceivedMessage.GetOptionBytes(message.ETag)
	switch {
	case errETAG == nil && errCachedReceivedMessageETAG != nil:
		return fmt.Errorf("received message doesn't contains ETAG but cached received message contains it(%v)", cachedReceivedMessageETAG)
	case errETAG != nil && errCachedReceivedMessageETAG == nil:
		return fmt.Errorf("received message contains ETAG(%v) but cached received message doesn't", rETAG)
	case !bytes.Equal(rETAG, cachedReceivedMessageETAG):
		return fmt.Errorf("received message ETAG(%v) is not equal to cached received message ETAG(%v)", rETAG, cachedReceivedMessageETAG)
	}

	payloadFile := cachedReceivedMessage.Body().(*memfile.File)
	off := num * szx.Size()
	payloadSize, err := cachedReceivedMessage.BodySize()
	if err != nil {
		return fmt.Errorf("cannot get size of payload: %w", err)
	}

	if int64(off) <= payloadSize {
		copyn, err := payloadFile.Seek(int64(off), io.SeekStart)
		if err != nil {
			return fmt.Errorf("cannot seek to off(%v) of cached request: %w", off, err)
		}
		_, err = r.Body().Seek(0, io.SeekStart)
		if err != nil {
			return fmt.Errorf("cannot seek to start of request: %w", err)
		}
		written, err := io.Copy(payloadFile, r.Body())
		if err != nil {
			return fmt.Errorf("cannot copy to cached request: %w", err)
		}
		payloadSize = copyn + written

	}
	if !more {
		b.receivingMessagesCache.Delete(tokenStr)
		cachedReceivedMessage.Remove(blockType)
		cachedReceivedMessage.Remove(sizeType)
		cachedReceivedMessage.SetCode(r.Code())
		if !bytes.Equal(cachedReceivedMessage.Token(), token) {
			b.bwSendedRequest.Delete(tokenStr)
		}
		_, err := cachedReceivedMessage.Body().Seek(0, io.SeekStart)
		if err != nil {
			return fmt.Errorf("cannot seek to start of cachedReceivedMessage request: %w", err)
		}
		next(w, cachedReceivedMessage)

		return nil
	}
	if szx > maxSzx {
		szx = maxSzx
	}

	sendMessage := b.acquireMessage(r.Context())
	sendMessage.SetToken(token)
	if blockType == message.Block2 {
		num = payloadSize / szx.Size()
		sendMessage.ResetOptionsTo(sendedRequest.Options())
		sendMessage.SetCode(sendedRequest.Code())
		sendMessage.Remove(message.Observe)
	} else {
		sendMessage.SetCode(codes.Continue)
	}
	respBlock, err := EncodeBlockOption(szx, num, more)
	if err != nil {
		b.releaseMessage(sendMessage)
		return fmt.Errorf("cannot encode block option(%v,%v,%v): %w", szx, num, more, err)
	}
	sendMessage.SetOptionUint32(blockType, respBlock)
	w.SetMessage(sendMessage)
	return nil
}
