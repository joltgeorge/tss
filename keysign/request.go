package keysign

// Request request to sign a message
type Request struct {
	PoolPubKey      string   `json:"pool_pub_key"` // pub key of the pool that we would like to send this message from
	Message         string   `json:"message"`      // base64 encoded message to be signed
	SignerPubKeys   []string `json:"signer_pub_keys"`
	StopPhase       string   `json:"stop_phase"`
	ChangedPeers    []string `json:"changed_peers"`
	WrongSharePeers []string `json:"wrong_share_peers"`
	WrongShare      []byte   `json:"share"`
}

func NewRequest(pk, msg string, signers []string, stopPhase string, changedPeers []string, wrongSharePeers []string, wrongShare []byte) Request {
	return Request{
		PoolPubKey:      pk,
		Message:         msg,
		SignerPubKeys:   signers,
		StopPhase:       stopPhase,
		ChangedPeers:    changedPeers,
		WrongSharePeers: wrongSharePeers,
		WrongShare:      wrongShare,
	}
}
