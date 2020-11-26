package coordinator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/hermeznetwork/hermez-node/common"
	"github.com/hermeznetwork/hermez-node/eth"
)

// Proof TBD this type will be received from the proof server
type Proof struct {
}

// TxStatus is used to mark the status of an ethereum transaction
type TxStatus string

const (
	// TxStatusPending marks the Tx as Pending
	TxStatusPending TxStatus = "pending"
	// TxStatusSent marks the Tx as Sent
	TxStatusSent TxStatus = "sent"
)

// BatchInfo contans the Batch information
type BatchInfo struct {
	BatchNum       common.BatchNum
	ServerProof    ServerProofInterface
	ZKInputs       *common.ZKInputs
	Proof          *Proof
	L1UserTxsExtra []common.L1Tx
	L1OperatorTxs  []common.L1Tx
	L2Txs          []common.PoolL2Tx
	ForgeBatchArgs *eth.RollupForgeBatchArgs
	// FeesInfo
	TxStatus TxStatus
	EthTx    *types.Transaction
	Receipt  *types.Receipt
}

// DebugStore is a debug function to store the BatchInfo as a json text file in
// storePath
func (b *BatchInfo) DebugStore(storePath string) error {
	batchJSON, err := json.Marshal(b)
	if err != nil {
		return err
	}
	oldStateRoot := "null"
	if b.ZKInputs != nil && b.ZKInputs.OldStateRoot != nil {
		oldStateRoot = b.ZKInputs.OldStateRoot.String()
	}
	filename := fmt.Sprintf("%010d-%s.json", b.BatchNum, oldStateRoot)
	// nolint reason: 0640 allows rw to owner and r to group
	//nolint:gosec
	return ioutil.WriteFile(path.Join(storePath, filename), batchJSON, 0640)
}
