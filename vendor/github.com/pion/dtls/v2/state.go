// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package dtls

import (
	"bytes"
	"encoding/gob"
	"sync/atomic"

	"github.com/pion/dtls/v2/pkg/crypto/elliptic"
	"github.com/pion/dtls/v2/pkg/crypto/prf"
	"github.com/pion/dtls/v2/pkg/protocol/handshake"
	"github.com/pion/transport/v3/replaydetector"
)

// State holds the dtls connection state and implements both encoding.BinaryMarshaler and encoding.BinaryUnmarshaler
type State struct {
	localEpoch, remoteEpoch   atomic.Value
	localSequenceNumber       []uint64 // uint48
	localRandom, remoteRandom handshake.Random
	masterSecret              []byte
	cipherSuite               CipherSuite // nil if a cipherSuite hasn't been chosen

	srtpProtectionProfile SRTPProtectionProfile // Negotiated SRTPProtectionProfile
	PeerCertificates      [][]byte
	IdentityHint          []byte
	SessionID             []byte

	// Connection Identifiers must be negotiated afresh on session resumption.
	// https://datatracker.ietf.org/doc/html/rfc9146#name-the-connection_id-extension

	// localConnectionID is the locally generated connection ID that is expected
	// to be received from the remote endpoint.
	// For a server, this is the connection ID sent in ServerHello.
	// For a client, this is the connection ID sent in the ClientHello.
	localConnectionID []byte
	// remoteConnectionID is the connection ID that the remote endpoint
	// specifies should be sent.
	// For a server, this is the connection ID received in the ClientHello.
	// For a client, this is the connection ID received in the ServerHello.
	remoteConnectionID []byte

	isClient bool

	preMasterSecret      []byte
	extendedMasterSecret bool

	namedCurve                 elliptic.Curve
	localKeypair               *elliptic.Keypair
	cookie                     []byte
	handshakeSendSequence      int
	handshakeRecvSequence      int
	serverName                 string
	remoteRequestedCertificate bool   // Did we get a CertificateRequest
	localCertificatesVerify    []byte // cache CertificateVerify
	localVerifyData            []byte // cached VerifyData
	localKeySignature          []byte // cached keySignature
	peerCertificatesVerified   bool

	replayDetector []replaydetector.ReplayDetector

	peerSupportedProtocols []string
	NegotiatedProtocol     string
}

type serializedState struct {
	LocalEpoch            uint16
	RemoteEpoch           uint16
	LocalRandom           [handshake.RandomLength]byte
	RemoteRandom          [handshake.RandomLength]byte
	CipherSuiteID         uint16
	MasterSecret          []byte
	SequenceNumber        uint64
	SRTPProtectionProfile uint16
	PeerCertificates      [][]byte
	IdentityHint          []byte
	SessionID             []byte
	LocalConnectionID     []byte
	RemoteConnectionID    []byte
	IsClient              bool
}

func (s *State) clone() *State {
	serialized := s.serialize()
	state := &State{}
	state.deserialize(*serialized)

	return state
}

func (s *State) serialize() *serializedState {
	// Marshal random values
	localRnd := s.localRandom.MarshalFixed()
	remoteRnd := s.remoteRandom.MarshalFixed()

	epoch := s.getLocalEpoch()
	return &serializedState{
		LocalEpoch:            s.getLocalEpoch(),
		RemoteEpoch:           s.getRemoteEpoch(),
		CipherSuiteID:         uint16(s.cipherSuite.ID()),
		MasterSecret:          s.masterSecret,
		SequenceNumber:        atomic.LoadUint64(&s.localSequenceNumber[epoch]),
		LocalRandom:           localRnd,
		RemoteRandom:          remoteRnd,
		SRTPProtectionProfile: uint16(s.srtpProtectionProfile),
		PeerCertificates:      s.PeerCertificates,
		IdentityHint:          s.IdentityHint,
		SessionID:             s.SessionID,
		LocalConnectionID:     s.localConnectionID,
		RemoteConnectionID:    s.remoteConnectionID,
		IsClient:              s.isClient,
	}
}

func (s *State) deserialize(serialized serializedState) {
	// Set epoch values
	epoch := serialized.LocalEpoch
	s.localEpoch.Store(serialized.LocalEpoch)
	s.remoteEpoch.Store(serialized.RemoteEpoch)

	for len(s.localSequenceNumber) <= int(epoch) {
		s.localSequenceNumber = append(s.localSequenceNumber, uint64(0))
	}

	// Set random values
	localRandom := &handshake.Random{}
	localRandom.UnmarshalFixed(serialized.LocalRandom)
	s.localRandom = *localRandom

	remoteRandom := &handshake.Random{}
	remoteRandom.UnmarshalFixed(serialized.RemoteRandom)
	s.remoteRandom = *remoteRandom

	s.isClient = serialized.IsClient

	// Set master secret
	s.masterSecret = serialized.MasterSecret

	// Set cipher suite
	s.cipherSuite = cipherSuiteForID(CipherSuiteID(serialized.CipherSuiteID), nil)

	atomic.StoreUint64(&s.localSequenceNumber[epoch], serialized.SequenceNumber)
	s.srtpProtectionProfile = SRTPProtectionProfile(serialized.SRTPProtectionProfile)

	// Set remote certificate
	s.PeerCertificates = serialized.PeerCertificates

	s.IdentityHint = serialized.IdentityHint

	// Set local and remote connection IDs
	s.localConnectionID = serialized.LocalConnectionID
	s.remoteConnectionID = serialized.RemoteConnectionID

	s.SessionID = serialized.SessionID
}

func (s *State) initCipherSuite() error {
	if s.cipherSuite.IsInitialized() {
		return nil
	}

	localRandom := s.localRandom.MarshalFixed()
	remoteRandom := s.remoteRandom.MarshalFixed()

	var err error
	if s.isClient {
		err = s.cipherSuite.Init(s.masterSecret, localRandom[:], remoteRandom[:], true)
	} else {
		err = s.cipherSuite.Init(s.masterSecret, remoteRandom[:], localRandom[:], false)
	}
	if err != nil {
		return err
	}
	return nil
}

// MarshalBinary is a binary.BinaryMarshaler.MarshalBinary implementation
func (s *State) MarshalBinary() ([]byte, error) {
	serialized := s.serialize()

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(*serialized); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalBinary is a binary.BinaryUnmarshaler.UnmarshalBinary implementation
func (s *State) UnmarshalBinary(data []byte) error {
	enc := gob.NewDecoder(bytes.NewBuffer(data))
	var serialized serializedState
	if err := enc.Decode(&serialized); err != nil {
		return err
	}

	s.deserialize(serialized)

	return s.initCipherSuite()
}

// ExportKeyingMaterial returns length bytes of exported key material in a new
// slice as defined in RFC 5705.
// This allows protocols to use DTLS for key establishment, but
// then use some of the keying material for their own purposes
func (s *State) ExportKeyingMaterial(label string, context []byte, length int) ([]byte, error) {
	if s.getLocalEpoch() == 0 {
		return nil, errHandshakeInProgress
	} else if len(context) != 0 {
		return nil, errContextUnsupported
	} else if _, ok := invalidKeyingLabels()[label]; ok {
		return nil, errReservedExportKeyingMaterial
	}

	localRandom := s.localRandom.MarshalFixed()
	remoteRandom := s.remoteRandom.MarshalFixed()

	seed := []byte(label)
	if s.isClient {
		seed = append(append(seed, localRandom[:]...), remoteRandom[:]...)
	} else {
		seed = append(append(seed, remoteRandom[:]...), localRandom[:]...)
	}
	return prf.PHash(s.masterSecret, seed, length, s.cipherSuite.HashFunc())
}

func (s *State) getRemoteEpoch() uint16 {
	if remoteEpoch, ok := s.remoteEpoch.Load().(uint16); ok {
		return remoteEpoch
	}
	return 0
}

func (s *State) getLocalEpoch() uint16 {
	if localEpoch, ok := s.localEpoch.Load().(uint16); ok {
		return localEpoch
	}
	return 0
}
