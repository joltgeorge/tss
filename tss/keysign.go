package tss

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/libp2p/go-libp2p-core/peer"

	"gitlab.com/thorchain/tss/go-tss/blame"
	"gitlab.com/thorchain/tss/go-tss/common"
	"gitlab.com/thorchain/tss/go-tss/conversion"
	"gitlab.com/thorchain/tss/go-tss/keysign"
	"gitlab.com/thorchain/tss/go-tss/messages"
	"gitlab.com/thorchain/tss/go-tss/p2p"
	"gitlab.com/thorchain/tss/go-tss/storage"
)

func (t *TssServer) waitForSignatures(msgID, poolPubKey string, msgToSign []byte, sigChan chan string) (keysign.Response, error) {
	// TSS keysign include both form party and keysign itself, thus we wait twice of the timeout
	data, err := t.signatureNotifier.WaitForSignature(msgID, msgToSign, poolPubKey, t.conf.KeySignTimeout, sigChan)
	if err != nil {
		return keysign.Response{}, err
	}

	if data == nil || (len(data.S) == 0 && len(data.R) == 0) {
		return keysign.Response{}, errors.New("keysign failed")
	}
	return keysign.NewResponse(
		base64.StdEncoding.EncodeToString(data.R),
		base64.StdEncoding.EncodeToString(data.S),
		common.Success,
		blame.Blame{},
	), nil
}

func (t *TssServer) generateSignature(msgID string, msgToSign []byte, req keysign.Request, threshold int, allSigners []peer.ID, localStateItem storage.KeygenLocalState, blameMgr *blame.Manager, keysignInstance *keysign.TssKeySign, sigChan chan string) (keysign.Response, error) {
	onlinePeers, err := t.joinParty(msgID, req.BlockHeight, req.SignerPubKeys, threshold, sigChan)
	if err != nil {
		// we received the signature from waiting for signature
		if errors.Is(err, p2p.ErrSignReceived) {
			return keysign.Response{}, err
		}
		if onlinePeers == nil {
			t.logger.Error().Err(err).Msg("error before we start join party")
			t.broadcastKeysignFailure(msgID, allSigners)
			return keysign.Response{
				Status: common.Fail,
				Blame:  blame.NewBlame(blame.InternalError, []blame.Node{}),
			}, nil
		}

		blameNodes, err := blameMgr.NodeSyncBlame(req.SignerPubKeys, onlinePeers)
		if err != nil {
			t.logger.Err(err).Msg("fail to get peers to blame")
		}
		t.broadcastKeysignFailure(msgID, allSigners)
		// make sure we blame the leader as well
		t.logger.Error().Err(err).Msgf("fail to form keysign party with online:%v", onlinePeers)
		return keysign.Response{
			Status: common.Fail,
			Blame:  blameNodes,
		}, nil

	}
	isKeySignMember := false
	for _, el := range onlinePeers {
		if el == t.p2pCommunication.GetHost().ID() {
			isKeySignMember = true
		}
	}
	if !isKeySignMember {
		// we are not the keysign member so we quit keysign and waiting for signature
		return keysign.Response{}, p2p.ErrSignReceived
	}

	parsedPeers := make([]string, len(onlinePeers))
	for i, el := range onlinePeers {
		parsedPeers[i] = el.String()
	}

	signers, err := conversion.GetPubKeysFromPeerIDs(parsedPeers)
	if err != nil {
		sigChan <- "signature generated"
		return keysign.Response{
			Status: common.Fail,
			Blame:  blame.Blame{},
		}, nil
	}

	signatureData, err := keysignInstance.SignMessage(msgToSign, localStateItem, signers)
	// the statistic of keygen only care about Tss it self, even if the following http response aborts,
	// it still counted as a successful keygen as the Tss model runs successfully.
	if err != nil {
		t.logger.Error().Err(err).Msg("err in keysign")
		sigChan <- "signature generated"
		atomic.AddUint64(&t.Status.FailedKeySign, 1)
		t.broadcastKeysignFailure(msgID, allSigners)
		blameNodes := *blameMgr.GetBlame()
		return keysign.Response{
			Status: common.Fail,
			Blame:  blameNodes,
		}, nil
	}

	atomic.AddUint64(&t.Status.SucKeySign, 1)

	sigChan <- "signature generated"
	// update signature notification
	if err := t.signatureNotifier.BroadcastSignature(msgID, signatureData, allSigners); err != nil {
		return keysign.Response{}, fmt.Errorf("fail to broadcast signature:%w", err)
	}
	return keysign.NewResponse(
		base64.StdEncoding.EncodeToString(signatureData.R),
		base64.StdEncoding.EncodeToString(signatureData.S),
		common.Success,
		blame.Blame{},
	), nil
}

func (t *TssServer) KeySign(req keysign.Request) (keysign.Response, error) {
	t.logger.Info().Str("pool pub key", req.PoolPubKey).
		Str("signer pub keys", strings.Join(req.SignerPubKeys, ",")).
		Str("msg", req.Message).
		Msg("received keysign request")
	emptyResp := keysign.Response{}
	msgID, err := t.requestToMsgId(req)
	if err != nil {
		return emptyResp, err
	}

	keysignInstance := keysign.NewTssKeySign(
		t.p2pCommunication.GetLocalPeerID(),
		t.conf,
		t.p2pCommunication.BroadcastMsgChan,
		t.stopChan,
		msgID,
		t.privateKey,
		t.p2pCommunication,
		t.stateManager,
	)

	keySignChannels := keysignInstance.GetTssKeySignChannels()
	t.p2pCommunication.SetSubscribe(messages.TSSKeySignMsg, msgID, keySignChannels)
	t.p2pCommunication.SetSubscribe(messages.TSSKeySignVerMsg, msgID, keySignChannels)
	t.p2pCommunication.SetSubscribe(messages.TSSControlMsg, msgID, keySignChannels)
	t.p2pCommunication.SetSubscribe(messages.TSSTaskDone, msgID, keySignChannels)

	defer func() {
		t.p2pCommunication.CancelSubscribe(messages.TSSKeySignMsg, msgID)
		t.p2pCommunication.CancelSubscribe(messages.TSSKeySignVerMsg, msgID)
		t.p2pCommunication.CancelSubscribe(messages.TSSControlMsg, msgID)
		t.p2pCommunication.CancelSubscribe(messages.TSSTaskDone, msgID)

		t.p2pCommunication.ReleaseStream(msgID)
		t.signatureNotifier.ReleaseStream(msgID)
		t.partyCoordinator.ReleaseStream(msgID)
	}()

	localStateItem, err := t.stateManager.GetLocalState(req.PoolPubKey)
	if err != nil {
		return emptyResp, fmt.Errorf("fail to get local keygen state: %w", err)
	}
	msgToSign, err := base64.StdEncoding.DecodeString(req.Message)
	if err != nil {
		return emptyResp, fmt.Errorf("fail to decode message(%s): %w", req.Message, err)
	}
	if len(req.SignerPubKeys) == 0 {
		return emptyResp, errors.New("empty signer pub keys")
	}

	threshold, err := conversion.GetThreshold(len(localStateItem.ParticipantKeys))
	if err != nil {
		t.logger.Error().Err(err).Msg("fail to get the threshold")
		return emptyResp, errors.New("fail to get threshold")
	}
	if len(req.SignerPubKeys) <= threshold {
		t.logger.Error().Msgf("not enough signers, threshold=%d and signers=%d", threshold, len(req.SignerPubKeys))
		return emptyResp, errors.New("not enough signers")
	}

	blameMgr := keysignInstance.GetTssCommonStruct().GetBlameMgr()
	// get all the tss nodes that were part of the original key gen
	allSigners, err := conversion.GetPeerIDs(localStateItem.ParticipantKeys)
	if err != nil {
		return emptyResp, fmt.Errorf("fail to convert pub keys to peer id:%w", err)
	}

	var receivedSig, generatedSig keysign.Response
	var errWait, errGen error
	sigChan := make(chan string, 2)
	wg := sync.WaitGroup{}
	wg.Add(2)
	// we wait for signatures
	go func() {
		defer wg.Done()
		receivedSig, errWait = t.waitForSignatures(msgID, req.PoolPubKey, msgToSign, sigChan)
		// we received an valid signature indeed
		if errWait == nil {
			sigChan <- "signature received"
		}
	}()

	// we generate the signature ourselves
	go func() {
		defer wg.Done()
		generatedSig, errGen = t.generateSignature(msgID, msgToSign, req, threshold, allSigners, localStateItem, blameMgr, keysignInstance, sigChan)
	}()
	wg.Wait()
	close(sigChan)

	// we received the generated verified signature, so we return
	if errWait == nil {
		return receivedSig, nil
	}
	// for this round, we are not the active signer
	if errors.Is(errGen, p2p.ErrSignReceived) {
		return receivedSig, nil
	}

	return generatedSig, errGen
}

func (t *TssServer) broadcastKeysignFailure(messageID string, peers []peer.ID) {
	if err := t.signatureNotifier.BroadcastFailed(messageID, peers); err != nil {
		t.logger.Err(err).Msg("fail to broadcast keysign failure")
	}
}

func (t *TssServer) isPartOfKeysignParty(parties []string) bool {
	for _, item := range parties {
		if t.localNodePubKey == item {
			return true
		}
	}
	return false
}
