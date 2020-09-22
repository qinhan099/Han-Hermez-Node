package historydb

import (
	"math/big"
	"time"

	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/hermeznetwork/hermez-node/common"
	"github.com/iden3/go-iden3-crypto/babyjub"
)

// HistoryTx is a representation of a generic Tx with additional information
// required by the API, and extracted by joining block and token tables
type HistoryTx struct {
	// Generic
	IsL1        bool            `meddler:"is_l1"`
	TxID        common.TxID     `meddler:"id"`
	Type        common.TxType   `meddler:"type"`
	Position    int             `meddler:"position"`
	FromIdx     common.Idx      `meddler:"from_idx"`
	ToIdx       common.Idx      `meddler:"to_idx"`
	Amount      *big.Int        `meddler:"amount,bigint"`
	AmountFloat float64         `meddler:"amount_f"`
	TokenID     common.TokenID  `meddler:"token_id"`
	USD         float64         `meddler:"amount_usd,zeroisnull"`
	BatchNum    common.BatchNum `meddler:"batch_num,zeroisnull"` // batchNum in which this tx was forged. If the tx is L2, this must be != 0
	EthBlockNum int64           `meddler:"eth_block_num"`        // Ethereum Block Number in which this L1Tx was added to the queue
	// L1
	ToForgeL1TxsNum int64              `meddler:"to_forge_l1_txs_num"` // toForgeL1TxsNum in which the tx was forged / will be forged
	UserOrigin      bool               `meddler:"user_origin"`         // true if the tx was originated by a user, false if it was aoriginated by a coordinator. Note that this differ from the spec for implementation simplification purpposes
	FromEthAddr     ethCommon.Address  `meddler:"from_eth_addr"`
	FromBJJ         *babyjub.PublicKey `meddler:"from_bjj"`
	LoadAmount      *big.Int           `meddler:"load_amount,bigintnull"`
	LoadAmountFloat float64            `meddler:"load_amount_f"`
	LoadAmountUSD   float64            `meddler:"load_amount_usd,zeroisnull"`
	// L2
	Fee    common.FeeSelector `meddler:"fee,zeroisnull"`
	FeeUSD float64            `meddler:"fee_usd,zeroisnull"`
	Nonce  common.Nonce       `meddler:"nonce,zeroisnull"`
	// API extras
	Timestamp   time.Time `meddler:"timestamp,utctime"`
	TokenSymbol string    `meddler:"symbol"`
	CurrentUSD  float64   `meddler:"current_usd"`
	USDUpdate   time.Time `meddler:"usd_update,utctime"`
	TotalItems  int       `meddler:"total_items"`
}