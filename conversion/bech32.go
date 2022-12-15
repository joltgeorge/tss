package conversion

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func SetupBech32Prefix() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("oppy", "oppypub")
	config.SetBech32PrefixForValidator("oppyval", "oppyvalpub")
	config.SetBech32PrefixForConsensusNode("oppyvalcons", "oppyvalconspub")
}
