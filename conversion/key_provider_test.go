package conversion

import (
	sdk256key "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/types/bech32/legacybech32"
	"testing"

	"github.com/stretchr/testify/assert"
	. "gopkg.in/check.v1"
)

const testPriKey = "OTI4OTdkYzFjMWFhMjU3MDNiMTE4MDM1OTQyY2Y3MDVkOWFhOGIzN2JlOGIwOWIwMTZjYTkxZjNjOTBhYjhlYQ=="

type KeyProviderTestSuite struct{}

var _ = Suite(&KeyProviderTestSuite{})

func TestGetPubKeysFromPeerIDs(t *testing.T) {
	SetupBech32Prefix()
	input := []string{
		"12D3KooWLaTQ4qDLJZtv6toT8ATS7rkLf8azxKqhsGt8ZxhoLgRX",
		"12D3KooWSVeLmfxFoBfGq38Yi8wdTZh7mrhmro9zYfVwWH8KEpcY",
	}
	result, err := GetPubKeysFromPeerIDs(input)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	assert.Len(t, result, 2)
	assert.Equal(t, "oppypub1zcjduepqnls9drdepkay59gr2jkyut06mwcff9ydadedluq4w6vw95zz8x3qv7hqs2", result[0])
	assert.Equal(t, "oppypub1zcjduepq7l9wmkvqvprdv0zmmeuwycjylx0rrju5gz20ae4ykfypagh632csprvxkt", result[1])
	input1 := append(input, "whatever")
	result, err = GetPubKeysFromPeerIDs(input1)
	assert.NotNil(t, err)
	assert.Nil(t, result)
}

func (*KeyProviderTestSuite) TestGetPriKey(c *C) {
	pk, err := GetPriKey("32whatever")
	c.Assert(err, NotNil)
	c.Assert(pk, IsNil)
	pk, err = GetPriKey(testPriKey)
	c.Assert(err, IsNil)
	c.Assert(pk, NotNil)
	//result, err := GetPriKeyRawBytes(pk)
	//c.Assert(err, IsNil)
	//c.Assert(result, NotNil)
	//c.Assert(result, HasLen, 32)
}

func (KeyProviderTestSuite) TestGetPeerIDs(c *C) {
	SetupBech32Prefix()
	pubKeys := []string{
		"oppypub1zcjduepqnls9drdepkay59gr2jkyut06mwcff9ydadedluq4w6vw95zz8x3qv7hqs2",
		"oppypub1zcjduepq7l9wmkvqvprdv0zmmeuwycjylx0rrju5gz20ae4ykfypagh632csprvxkt",
	}
	peers, err := GetPeerIDs(pubKeys)
	c.Assert(err, IsNil)
	c.Assert(peers, NotNil)
	c.Assert(peers, HasLen, 2)
	c.Assert(peers[0].String(), Equals, "12D3KooWLaTQ4qDLJZtv6toT8ATS7rkLf8azxKqhsGt8ZxhoLgRX")
	c.Assert(peers[1].String(), Equals, "12D3KooWSVeLmfxFoBfGq38Yi8wdTZh7mrhmro9zYfVwWH8KEpcY")
	pubKeys1 := append(pubKeys, "helloworld")
	peers, err = GetPeerIDs(pubKeys1)
	c.Assert(err, NotNil)
	c.Assert(peers, IsNil)
}

func (KeyProviderTestSuite) TestGetPeerIDFromPubKey(c *C) {
	SetupBech32Prefix()
	pID, err := GetPeerIDFromPubKey("oppypub1zcjduepqnls9drdepkay59gr2jkyut06mwcff9ydadedluq4w6vw95zz8x3qv7hqs2")
	c.Assert(err, IsNil)
	c.Assert(pID.String(), Equals, "12D3KooWLaTQ4qDLJZtv6toT8ATS7rkLf8azxKqhsGt8ZxhoLgRX")
	pID1, err := GetPeerIDFromPubKey("whatever")
	c.Assert(err, NotNil)
	c.Assert(pID1.String(), Equals, "")
}

func (KeyProviderTestSuite) TestCheckKeyOnCurve(c *C) {
	_, err := CheckKeyOnCurve("aa")
	c.Assert(err, NotNil)
	SetupBech32Prefix()
	sk := sdk256key.GenPrivKey()
	pk, _ := legacybech32.MarshalPubKey(legacybech32.AccPK, sk.PubKey())
	_, err = CheckKeyOnCurve(pk)
	c.Assert(err, IsNil)
}
