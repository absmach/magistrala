package blockwise

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/dsnet/golib/memfile"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/net/responsewriter"
	"github.com/plgd-dev/go-coap/v3/pkg/cache"
	"golang.org/x/sync/semaphore"
)

// Block Option value is represented: https://tools.ietf.org/html/rfc7959#section-2.2
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
	// SZX16 block of size 16bytes
	SZX16 SZX = 0
	// SZX32 block of size 32bytes
	SZX32 SZX = 1
	// SZX64 block of size 64bytes
	SZX64 SZX = 2
	// SZX128 block of size 128bytes
	SZX128 SZX = 3
	// SZX256 block of size 256bytes
	SZX256 SZX = 4
	// SZX512 block of size 512bytes
	SZX512 SZX = 5
	// SZX1024 block of size 1024bytes
	SZX1024 SZX = 6
	// SZXBERT block of size n*1024bytes
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

	szx = SZX(blockVal & szxMask)                  // masking for the SZX
	if (blockVal & moreBlocksFollowingMask) != 0 { // masking for the "M"
		moreBlocksFollowing = true
	}
	blockNumber = int64(blockVal) >> 4 // shifting out the SZX and M vals. leaving the block number behind
	if blockNumber > maxBlockNumber {
		err = ErrBlockNumberExceedLimit
	}
	return
}

type Client interface {
	// create message from pool
	AcquireMessage(ctx context.Context) *pool.Message
	// return back the message to the pool for next use
	ReleaseMessage(m *pool.Message)
}

type BlockWise[C Client] struct {
	cc                        C
	receivingMessagesCache    *cache.Cache[uint64, *messageGuard]
	sendingMessagesCache      *cache.Cache[uint64, *pool.Message]
	errors                    func(error)
	getSentRequestFromOutside func(token message.Token) (*pool.Message, bool)
	expiration                time.Duration
}

type messageGuard struct {
	*pool.Message
	*semaphore.Weighted
}

func newRequestGuard(request *pool.Message) *messageGuard {
	return &messageGuard{
		Message:  request,
		Weighted: semaphore.NewWeighted(1),
	}
}

// New provides blockwise.
// getSentRequestFromOutside must returns a copy of request which will be released after use.
func New[C Client](
	cc C,
	expiration time.Duration,
	errors func(error),
	getSentRequestFromOutside func(token message.Token) (*pool.Message, bool),
) *BlockWise[C] {
	if getSentRequestFromOutside == nil {
		getSentRequestFromOutside = func(token message.Token) (*pool.Message, bool) { return nil, false }
	}
	return &BlockWise[C]{
		cc:                        cc,
		receivingMessagesCache:    cache.NewCache[uint64, *messageGuard](),
		sendingMessagesCache:      cache.NewCache[uint64, *pool.Message](),
		errors:                    errors,
		getSentRequestFromOutside: getSentRequestFromOutside,
		expiration:                expiration,
	}
}

func bufferSize(szx SZX, maxMessageSize uint32) int64 {
	if szx < SZXBERT {
		return szx.Size()
	}
	return (int64(maxMessageSize) / szx.Size()) * szx.Size()
}

// CheckExpirations iterates over caches and remove expired items.
func (b *BlockWise[C]) CheckExpirations(now time.Time) {
	b.receivingMessagesCache.CheckExpirations(now)
	b.sendingMessagesCache.CheckExpirations(now)
}

func (b *BlockWise[C]) cloneMessage(r *pool.Message) *pool.Message {
	req := b.cc.AcquireMessage(r.Context())
	req.SetCode(r.Code())
	req.SetToken(r.Token())
	req.ResetOptionsTo(r.Options())
	req.SetType(r.Type())
	return req
}

func payloadSizeError(err error) error {
	return fmt.Errorf("cannot get size of payload: %w", err)
}

// Do sends an coap message and returns an coap response via blockwise transfer.
func (b *BlockWise[C]) Do(r *pool.Message, maxSzx SZX, maxMessageSize uint32, do func(req *pool.Message) (*pool.Message, error)) (*pool.Message, error) {
	if maxSzx > SZXBERT {
		return nil, fmt.Errorf("invalid szx")
	}
	if len(r.Token()) == 0 {
		return nil, fmt.Errorf("invalid token")
	}

	expire, ok := r.Context().Deadline()
	if !ok {
		expire = time.Now().Add(b.expiration)
	}
	_, loaded := b.sendingMessagesCache.LoadOrStore(r.Token().Hash(), cache.NewElement(r, expire, nil))
	if loaded {
		return nil, fmt.Errorf("invalid token")
	}
	defer b.sendingMessagesCache.Delete(r.Token().Hash())
	if r.Body() == nil {
		return do(r)
	}
	payloadSize, err := r.BodySize()
	if err != nil {
		return nil, payloadSizeError(err)
	}
	if payloadSize <= maxSzx.Size() {
		return do(r)
	}

	switch r.Code() {
	case codes.POST, codes.PUT:
		break
	default:
		return nil, fmt.Errorf("unsupported command(%v)", r.Code())
	}
	req := b.cloneMessage(r)
	defer b.cc.ReleaseMessage(req)
	req.SetOptionUint32(message.Size1, uint32(payloadSize))
	block, err := EncodeBlockOption(maxSzx, 0, true)
	if err != nil {
		return nil, fmt.Errorf("cannot encode block option(%v, %v, %v) to bw request: %w", maxSzx, 0, true, err)
	}
	req.SetOptionUint32(message.Block1, block)
	newBufLen := bufferSize(maxSzx, maxMessageSize)
	buf := make([]byte, newBufLen)
	newOff, err := r.Body().Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("cannot seek in payload: %w", err)
	}
	readed, err := io.ReadFull(r.Body(), buf)
	if errors.Is(err, io.ErrUnexpectedEOF) {
		if newOff+int64(readed) == payloadSize {
			err = nil
		}
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read payload: %w", err)
	}
	buf = buf[:readed]
	req.SetBody(bytes.NewReader(buf))
	return do(req)
}

func newWriteRequestResponse[C Client](cc C, request *pool.Message) *responsewriter.ResponseWriter[C] {
	req := cc.AcquireMessage(request.Context())
	req.SetCode(request.Code())
	req.SetToken(request.Token())
	req.ResetOptionsTo(request.Options())
	req.SetBody(request.Body())
	return responsewriter.New(req, cc, request.Options()...)
}

// WriteMessage sends an coap message via blockwise transfer.
func (b *BlockWise[C]) WriteMessage(request *pool.Message, maxSZX SZX, maxMessageSize uint32, writeMessage func(r *pool.Message) error) error {
	startSendingMessageBlock, err := EncodeBlockOption(maxSZX, 0, true)
	if err != nil {
		return fmt.Errorf("cannot encode start sending message block option(%v,%v,%v): %w", maxSZX, 0, true, err)
	}

	w := newWriteRequestResponse(b.cc, request)
	err = b.startSendingMessage(w, maxSZX, maxMessageSize, startSendingMessageBlock)
	if err != nil {
		return fmt.Errorf("cannot start writing request: %w", err)
	}
	return writeMessage(w.Message())
}

func fitSZX(r *pool.Message, blockType message.OptionID, maxSZX SZX) SZX {
	block, err := r.GetOptionUint32(blockType)
	if err != nil {
		return maxSZX
	}

	szx, _, _, err := DecodeBlockOption(block)
	if err != nil {
		return maxSZX
	}

	if maxSZX > szx {
		return szx
	}
	return maxSZX
}

func (b *BlockWise[C]) sendEntityIncomplete(w *responsewriter.ResponseWriter[C], token message.Token) {
	sendMessage := b.cc.AcquireMessage(w.Message().Context())
	sendMessage.SetCode(codes.RequestEntityIncomplete)
	sendMessage.SetToken(token)
	sendMessage.SetType(message.NonConfirmable)
	w.SetMessage(sendMessage)
}

func wantsToBeReceived(r *pool.Message) bool {
	hasBlock1 := r.HasOption(message.Block1)
	hasBlock2 := r.HasOption(message.Block2)
	if hasBlock1 && (r.Code() == codes.POST || r.Code() == codes.PUT) {
		// r contains payload which we received
		return true
	}
	if hasBlock2 && (r.Code() >= codes.GET && r.Code() <= codes.DELETE) {
		// r is command to get next block
		return false
	}
	if r.Code() == codes.Continue {
		return false
	}
	return true
}

func (b *BlockWise[C]) getSendingMessageCode(token uint64) (codes.Code, bool) {
	v := b.sendingMessagesCache.Load(token)
	if v == nil {
		return codes.Empty, false
	}
	return v.Data().Code(), true
}

// Handle middleware which constructs COAP request from blockwise transfer and send COAP response via blockwise.
func (b *BlockWise[C]) Handle(w *responsewriter.ResponseWriter[C], r *pool.Message, maxSZX SZX, maxMessageSize uint32, next func(w *responsewriter.ResponseWriter[C], r *pool.Message)) {
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
	tokenStr := token.Hash()

	sendingMessageCode, sendingMessageExist := b.getSendingMessageCode(tokenStr)
	if !sendingMessageExist || wantsToBeReceived(r) {
		err := b.handleReceivedMessage(w, r, maxSZX, maxMessageSize, next)
		if err != nil {
			b.sendEntityIncomplete(w, token)
			b.errors(fmt.Errorf("handleReceivedMessage(%v): %w", r, err))
		}
		return
	}
	more, err := b.continueSendingMessage(w, r, maxSZX, maxMessageSize, sendingMessageCode)
	if err != nil {
		b.sendingMessagesCache.Delete(tokenStr)
		b.errors(fmt.Errorf("continueSendingMessage(%v): %w", r, err))
		return
	}
	// For codes GET,POST,PUT,DELETE, we want them to wait for pairing response and then delete them when the full response comes in or when timeout occurs.
	if !more && sendingMessageCode > codes.DELETE {
		b.sendingMessagesCache.Delete(tokenStr)
	}
}

func (b *BlockWise[C]) handleReceivedMessage(w *responsewriter.ResponseWriter[C], r *pool.Message, maxSZX SZX, maxMessageSize uint32, next func(w *responsewriter.ResponseWriter[C], r *pool.Message)) error {
	startSendingMessageBlock, err := EncodeBlockOption(maxSZX, 0, true)
	if err != nil {
		return fmt.Errorf("cannot encode start sending message block option(%v,%v,%v): %w", maxSZX, 0, true, err)
	}
	switch r.Code() {
	case codes.Empty, codes.CSM, codes.Ping, codes.Pong, codes.Release, codes.Abort:
		next(w, r)
		return nil
	case codes.GET, codes.DELETE:
		maxSZX = fitSZX(r, message.Block2, maxSZX)
		block, errG := r.GetOptionUint32(message.Block2)
		if errG == nil {
			r.Remove(message.Block2)
		}
		next(w, r)
		if w.Message().Code() == codes.Content && errG == nil {
			startSendingMessageBlock = block
		}
	case codes.POST, codes.PUT:
		maxSZX = fitSZX(r, message.Block1, maxSZX)
		errP := b.processReceivedMessage(w, r, maxSZX, next, message.Block1, message.Size1)
		if errP != nil {
			return errP
		}
	default:
		maxSZX = fitSZX(r, message.Block2, maxSZX)
		errP := b.processReceivedMessage(w, r, maxSZX, next, message.Block2, message.Size2)
		if errP != nil {
			return errP
		}
	}
	return b.startSendingMessage(w, maxSZX, maxMessageSize, startSendingMessageBlock)
}

func (b *BlockWise[C]) createSendingMessage(sendingMessage *pool.Message, maxSZX SZX, maxMessageSize uint32, block uint32) (sendMessage *pool.Message, more bool, err error) {
	blockType := message.Block2
	sizeType := message.Size2
	token := sendingMessage.Token()
	switch sendingMessage.Code() {
	case codes.POST, codes.PUT:
		blockType = message.Block1
		sizeType = message.Size1
	}

	szx, num, _, err := DecodeBlockOption(block)
	if err != nil {
		return nil, false, fmt.Errorf("cannot decode %v option: %w", blockType, err)
	}

	sendMessage = b.cc.AcquireMessage(sendingMessage.Context())
	sendMessage.SetCode(sendingMessage.Code())
	sendMessage.ResetOptionsTo(sendingMessage.Options())
	sendMessage.SetToken(token)
	sendMessage.SetType(sendingMessage.Type())
	payloadSize, err := sendingMessage.BodySize()
	if err != nil {
		b.cc.ReleaseMessage(sendMessage)
		return nil, false, payloadSizeError(err)
	}
	if szx > maxSZX {
		szx = maxSZX
	}
	newBufLen := bufferSize(szx, maxMessageSize)
	off := num * szx.Size()
	if blockType == message.Block1 {
		// For block1, we need to skip the already sent bytes.
		off += newBufLen
	}
	offSeek, err := sendingMessage.Body().Seek(off, io.SeekStart)
	if err != nil {
		b.cc.ReleaseMessage(sendMessage)
		return nil, false, fmt.Errorf("cannot seek in response: %w", err)
	}
	if off != offSeek {
		b.cc.ReleaseMessage(sendMessage)
		return nil, false, fmt.Errorf("cannot seek to requested offset(%v != %v)", off, offSeek)
	}
	buf := make([]byte, 1024)
	if int64(len(buf)) < newBufLen {
		buf = make([]byte, newBufLen)
	}
	buf = buf[:newBufLen]

	readed, err := io.ReadFull(sendingMessage.Body(), buf)
	if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		if offSeek+int64(readed) == payloadSize {
			err = nil
		}
	}
	if err != nil {
		b.cc.ReleaseMessage(sendMessage)
		return nil, false, fmt.Errorf("cannot read response: %w", err)
	}

	buf = buf[:readed]
	sendMessage.SetBody(bytes.NewReader(buf))
	more = true
	if offSeek+int64(readed) == payloadSize {
		more = false
	}
	sendMessage.SetOptionUint32(sizeType, uint32(payloadSize))
	num = (offSeek) / szx.Size()
	block, err = EncodeBlockOption(szx, num, more)
	if err != nil {
		b.cc.ReleaseMessage(sendMessage)
		return nil, false, fmt.Errorf("cannot encode block option(%v,%v,%v): %w", szx, num, more, err)
	}
	sendMessage.SetOptionUint32(blockType, block)
	return sendMessage, more, nil
}

func (b *BlockWise[C]) continueSendingMessage(w *responsewriter.ResponseWriter[C], r *pool.Message, maxSZX SZX, maxMessageSize uint32, sendingMessageCode codes.Code /* msg *pool.Message*/) (bool, error) {
	blockType := message.Block2
	switch sendingMessageCode {
	case codes.POST, codes.PUT:
		blockType = message.Block1
	}

	block, err := r.GetOptionUint32(blockType)
	if err != nil {
		return false, fmt.Errorf("cannot get %v option: %w", blockType, err)
	}
	var sendMessage *pool.Message
	var more bool
	b.sendingMessagesCache.LoadWithFunc(r.Token().Hash(), func(value *cache.Element[*pool.Message]) *cache.Element[*pool.Message] {
		sendMessage, more, err = b.createSendingMessage(value.Data(), maxSZX, maxMessageSize, block)
		if err != nil {
			err = fmt.Errorf("cannot create sending message: %w", err)
		}
		return nil
	})
	if err == nil && sendMessage == nil {
		err = fmt.Errorf("cannot find sending message for token(%v)", r.Token())
	}
	if err != nil {
		return false, fmt.Errorf("handleSendingMessage: %w", err)
	}
	w.SetMessage(sendMessage)
	return more, err
}

func isObserveResponse(msg *pool.Message) bool {
	_, err := msg.GetOptionUint32(message.Observe)
	if err != nil {
		return false
	}
	return msg.Code() >= codes.Created
}

func (b *BlockWise[C]) startSendingMessage(w *responsewriter.ResponseWriter[C], maxSZX SZX, maxMessageSize uint32, block uint32) error {
	payloadSize, err := w.Message().BodySize()
	if err != nil {
		return payloadSizeError(err)
	}

	if payloadSize < maxSZX.Size() {
		return nil
	}
	sendingMessage, _, err := b.createSendingMessage(w.Message(), maxSZX, maxMessageSize, block)
	if err != nil {
		return fmt.Errorf("handleSendingMessage: cannot create sending message: %w", err)
	}
	originalSendingMessage := w.Swap(sendingMessage)
	if isObserveResponse(w.Message()) {
		b.cc.ReleaseMessage(originalSendingMessage)
		// https://tools.ietf.org/html/rfc7959#section-2.6 - we don't need store it because client will be get values via GET.
		return nil
	}
	expire, ok := sendingMessage.Context().Deadline()
	if !ok {
		expire = time.Now().Add(b.expiration)
	}
	el, loaded := b.sendingMessagesCache.LoadOrStore(sendingMessage.Token().Hash(), cache.NewElement(originalSendingMessage, expire, nil))
	if loaded {
		defer b.cc.ReleaseMessage(originalSendingMessage)
		return fmt.Errorf("cannot add message (%v) to sending message cache: message(%v) with token(%v) already exist", originalSendingMessage, el.Data(), sendingMessage.Token())
	}
	return nil
}

func (b *BlockWise[C]) getSentRequest(token message.Token) *pool.Message {
	data, ok := b.sendingMessagesCache.LoadWithFunc(token.Hash(), func(value *cache.Element[*pool.Message]) *cache.Element[*pool.Message] {
		if value == nil {
			return nil
		}
		v := value.Data()
		msg := b.cc.AcquireMessage(v.Context())
		msg.SetCode(v.Code())
		msg.SetToken(v.Token())
		msg.ResetOptionsTo(v.Options())
		msg.SetType(v.Type())
		return cache.NewElement(msg, value.ValidUntil.Load(), nil)
	})
	if ok {
		return data.Data()
	}
	globalRequest, ok := b.getSentRequestFromOutside(token)
	if ok {
		return globalRequest
	}
	return nil
}

func (b *BlockWise[C]) handleObserveResponse(sentRequest *pool.Message) (message.Token, time.Time, error) {
	// https://tools.ietf.org/html/rfc7959#section-2.6 - performs GET with new token.
	if sentRequest == nil {
		return nil, time.Time{}, fmt.Errorf("observation is not registered")
	}
	token, err := message.GetToken()
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("cannot get token for create GET request: %w", err)
	}
	validUntil := time.Now().Add(b.expiration) // context of observation can be expired.
	bwSentRequest := b.cloneMessage(sentRequest)
	bwSentRequest.SetToken(token)
	_, loaded := b.sendingMessagesCache.LoadOrStore(token.Hash(), cache.NewElement(bwSentRequest, validUntil, nil))
	if loaded {
		return nil, time.Time{}, fmt.Errorf("cannot process message: message with token already exist")
	}
	return token, validUntil, nil
}

func (b *BlockWise[C]) getValidUntil(sentRequest *pool.Message) time.Time {
	validUntil := time.Now().Add(b.expiration)
	if sentRequest != nil {
		if deadline, ok := sentRequest.Context().Deadline(); ok {
			return deadline
		}
	}
	return validUntil
}

func getSzx(szx, maxSzx SZX) SZX {
	if szx > maxSzx {
		return maxSzx
	}
	return szx
}

func (b *BlockWise[C]) getPayloadFromCachedReceivedMessage(r, cachedReceivedMessage *pool.Message) (*memfile.File, int64, error) {
	payloadFile, ok := cachedReceivedMessage.Body().(*memfile.File)
	if !ok {
		return nil, 0, fmt.Errorf("invalid body type(%T) stored in receivingMessagesCache", cachedReceivedMessage.Body())
	}
	rETAG, errETAG := r.GetOptionBytes(message.ETag)
	cachedReceivedMessageETAG, errCachedReceivedMessageETAG := cachedReceivedMessage.GetOptionBytes(message.ETag)
	switch {
	case errETAG == nil && errCachedReceivedMessageETAG != nil:
		if len(cachedReceivedMessageETAG) > 0 { // make sure there is an etag there
			return nil, 0, fmt.Errorf("received message doesn't contains ETAG but cached received message contains it(%v)", cachedReceivedMessageETAG)
		}
	case errETAG != nil && errCachedReceivedMessageETAG == nil:
		if len(rETAG) > 0 { // make sure there is an etag there
			return nil, 0, fmt.Errorf("received message contains ETAG(%v) but cached received message doesn't", rETAG)
		}
	case !bytes.Equal(rETAG, cachedReceivedMessageETAG):
		// ETAG was changed - drop data and set new ETAG
		cachedReceivedMessage.SetOptionBytes(message.ETag, rETAG)
		if err := payloadFile.Truncate(0); err != nil {
			return nil, 0, fmt.Errorf("cannot truncate cached request: %w", err)
		}
	}

	payloadSize, err := cachedReceivedMessage.BodySize()
	if err != nil {
		return nil, 0, payloadSizeError(err)
	}
	return payloadFile, payloadSize, nil
}

func copyToPayloadFromOffset(r *pool.Message, payloadFile *memfile.File, offset int64) (int64, error) {
	payloadSize := int64(0)
	copyn, err := payloadFile.Seek(offset, io.SeekStart)
	if err != nil {
		return 0, fmt.Errorf("cannot seek to off(%v) of cached request: %w", offset, err)
	}
	written := int64(0)
	if r.Body() != nil {
		_, err = r.Body().Seek(0, io.SeekStart)
		if err != nil {
			return 0, fmt.Errorf("cannot seek to start of request: %w", err)
		}
		written, err = io.Copy(payloadFile, r.Body())
		if err != nil {
			return 0, fmt.Errorf("cannot copy to cached request: %w", err)
		}
	}
	payloadSize = copyn + written
	err = payloadFile.Truncate(payloadSize)
	if err != nil {
		return 0, fmt.Errorf("cannot truncate cached request: %w", err)
	}
	return payloadSize, nil
}

func (b *BlockWise[C]) getCachedReceivedMessage(mg *messageGuard, r *pool.Message, tokenStr uint64, validUntil time.Time) (*pool.Message, func(), error) {
	cannotLockError := func(err error) error {
		return fmt.Errorf("processReceivedMessage: cannot lock message: %w", err)
	}
	if mg != nil {
		errA := mg.Acquire(mg.Context(), 1)
		if errA != nil {
			return nil, nil, cannotLockError(errA)
		}
		return mg.Message, func() { mg.Release(1) }, nil
	}
	closeFnList := []func(){}
	appendToClose := func(m *messageGuard) {
		closeFnList = append(closeFnList, func() {
			m.Release(1)
		})
	}
	closeFn := func() {
		for i := range closeFnList {
			closeFnList[len(closeFnList)-1-i]()
		}
	}
	msg := b.cc.AcquireMessage(r.Context())
	msg.ResetOptionsTo(r.Options())
	msg.SetToken(r.Token())
	msg.SetSequence(r.Sequence())
	msg.SetBody(memfile.New(make([]byte, 0, 1024)))
	msg.SetCode(r.Code())
	mg = newRequestGuard(msg)
	errA := mg.Acquire(mg.Context(), 1)
	if errA != nil {
		return nil, nil, cannotLockError(errA)
	}
	appendToClose(mg)
	element, loaded := b.receivingMessagesCache.LoadOrStore(tokenStr, cache.NewElement(mg, validUntil, func(d *messageGuard) {
		if d == nil {
			return
		}
		b.sendingMessagesCache.Delete(tokenStr)
	}))
	// request was already stored in cache, silently
	if loaded {
		mg = element.Data()
		if mg == nil {
			closeFn()
			return nil, nil, fmt.Errorf("request was already stored in cache")
		}
		errA := mg.Acquire(mg.Context(), 1)
		if errA != nil {
			closeFn()
			return nil, nil, cannotLockError(errA)
		}
		appendToClose(mg)
	}

	return mg.Message, closeFn, nil
}

//nolint:gocyclo,gocognit
func (b *BlockWise[C]) processReceivedMessage(w *responsewriter.ResponseWriter[C], r *pool.Message, maxSzx SZX, next func(w *responsewriter.ResponseWriter[C], r *pool.Message), blockType message.OptionID, sizeType message.OptionID) error {
	// TODO: lower cyclomatic complexity
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
		if errors.Is(err, message.ErrOptionNotFound) {
			next(w, r)
			return nil
		}
		return fmt.Errorf("cannot get Block(optionID=%d) option: %w", blockType, err)
	}
	szx, num, more, err := DecodeBlockOption(block)
	if err != nil {
		return fmt.Errorf("cannot decode block option: %w", err)
	}
	sentRequest := b.getSentRequest(token)
	if sentRequest != nil {
		defer b.cc.ReleaseMessage(sentRequest)
	}
	validUntil := b.getValidUntil(sentRequest)
	if blockType == message.Block2 && sentRequest == nil {
		return fmt.Errorf("cannot request body without paired request")
	}
	if isObserveResponse(r) {
		token, validUntil, err = b.handleObserveResponse(sentRequest)
		if err != nil {
			return fmt.Errorf("cannot process message: %w", err)
		}
	}

	tokenStr := token.Hash()
	var cachedReceivedMessageGuard *messageGuard
	if e := b.receivingMessagesCache.Load(tokenStr); e != nil {
		cachedReceivedMessageGuard = e.Data()
	}
	if cachedReceivedMessageGuard == nil {
		szx = getSzx(szx, maxSzx)
		// if there is no more then just forward req to next handler
		if !more {
			next(w, r)
			return nil
		}
	}
	cachedReceivedMessage, closeCachedReceivedMessage, err := b.getCachedReceivedMessage(cachedReceivedMessageGuard, r, tokenStr, validUntil)
	if err != nil {
		return err
	}
	defer closeCachedReceivedMessage()

	defer func(err *error) {
		if *err != nil {
			b.receivingMessagesCache.Delete(tokenStr)
		}
	}(&err)
	payloadFile, payloadSize, err := b.getPayloadFromCachedReceivedMessage(r, cachedReceivedMessage)
	if err != nil {
		return fmt.Errorf("cannot get payload: %w", err)
	}
	off := num * szx.Size()
	if off == payloadSize {
		payloadSize, err = copyToPayloadFromOffset(r, payloadFile, off)
		if err != nil {
			return fmt.Errorf("cannot copy data to payload: %w", err)
		}
		if !more {
			b.receivingMessagesCache.Delete(tokenStr)
			cachedReceivedMessage.Remove(blockType)
			cachedReceivedMessage.Remove(sizeType)
			cachedReceivedMessage.SetType(r.Type())
			if !bytes.Equal(cachedReceivedMessage.Token(), token) {
				b.sendingMessagesCache.Delete(tokenStr)
			}
			_, errS := cachedReceivedMessage.Body().Seek(0, io.SeekStart)
			if errS != nil {
				return fmt.Errorf("cannot seek to start of cachedReceivedMessage request: %w", errS)
			}
			next(w, cachedReceivedMessage)
			return nil
		}
	}

	szx = getSzx(szx, maxSzx)
	sendMessage := b.cc.AcquireMessage(r.Context())
	sendMessage.SetToken(token)
	if blockType == message.Block2 {
		num = payloadSize / szx.Size()
		sendMessage.ResetOptionsTo(sentRequest.Options())
		sendMessage.SetCode(sentRequest.Code())
		sendMessage.Remove(message.Observe)
		sendMessage.Remove(message.Block1)
		sendMessage.Remove(message.Size1)
	} else {
		sendMessage.SetCode(codes.Continue)
	}
	respBlock, err := EncodeBlockOption(szx, num, more)
	if err != nil {
		b.cc.ReleaseMessage(sendMessage)
		return fmt.Errorf("cannot encode block option(%v,%v,%v): %w", szx, num, more, err)
	}
	sendMessage.SetOptionUint32(blockType, respBlock)
	w.SetMessage(sendMessage)
	return nil
}
