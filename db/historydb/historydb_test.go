package historydb

import (
	"math"
	"math/big"
	"os"
	"testing"

	"github.com/hermeznetwork/hermez-node/common"
	dbUtils "github.com/hermeznetwork/hermez-node/db"
	"github.com/hermeznetwork/hermez-node/log"
	"github.com/hermeznetwork/hermez-node/test"
	"github.com/hermeznetwork/hermez-node/test/til"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var historyDB *HistoryDB

// In order to run the test you need to run a Posgres DB with
// a database named "history" that is accessible by
// user: "hermez"
// pass: set it using the env var POSTGRES_PASS
// This can be achieved by running: POSTGRES_PASS=your_strong_pass && sudo docker run --rm --name hermez-db-test -p 5432:5432 -e POSTGRES_DB=history -e POSTGRES_USER=hermez -e POSTGRES_PASSWORD=$POSTGRES_PASS -d postgres && sleep 2s && sudo docker exec -it hermez-db-test psql -a history -U hermez -c "CREATE DATABASE l2;"
// After running the test you can stop the container by running: sudo docker kill hermez-db-test
// If you already did that for the L2DB you don't have to do it again

func TestMain(m *testing.M) {
	// init DB
	pass := os.Getenv("POSTGRES_PASS")
	db, err := dbUtils.InitSQLDB(5432, "localhost", "hermez", pass, "hermez")
	if err != nil {
		panic(err)
	}
	historyDB = NewHistoryDB(db)
	if err != nil {
		panic(err)
	}
	// Run tests
	result := m.Run()
	// Close DB
	if err := db.Close(); err != nil {
		log.Error("Error closing the history DB:", err)
	}
	os.Exit(result)
}

func TestBlocks(t *testing.T) {
	var fromBlock, toBlock int64
	fromBlock = 1
	toBlock = 5
	// Delete peviously created rows (clean previous test execs)
	test.WipeDB(historyDB.DB())
	// Generate fake blocks
	blocks := test.GenBlocks(fromBlock, toBlock)
	// Insert blocks into DB
	for i := 0; i < len(blocks); i++ {
		err := historyDB.AddBlock(&blocks[i])
		assert.NoError(t, err)
	}
	// Get all blocks from DB
	fetchedBlocks, err := historyDB.GetBlocks(fromBlock, toBlock)
	assert.Equal(t, len(blocks), len(fetchedBlocks))
	// Compare generated vs getted blocks
	assert.NoError(t, err)
	for i := range fetchedBlocks {
		assertEqualBlock(t, &blocks[i], &fetchedBlocks[i])
	}
	// Get blocks from the DB one by one
	for i := fromBlock; i < toBlock; i++ {
		fetchedBlock, err := historyDB.GetBlock(i)
		assert.NoError(t, err)
		assertEqualBlock(t, &blocks[i-1], fetchedBlock)
	}
	// Get last block
	lastBlock, err := historyDB.GetLastBlock()
	assert.NoError(t, err)
	assertEqualBlock(t, &blocks[len(blocks)-1], lastBlock)
}

func assertEqualBlock(t *testing.T, expected *common.Block, actual *common.Block) {
	assert.Equal(t, expected.EthBlockNum, actual.EthBlockNum)
	assert.Equal(t, expected.Hash, actual.Hash)
	assert.Equal(t, expected.Timestamp.Unix(), actual.Timestamp.Unix())
}

func TestBatches(t *testing.T) {
	const fromBlock int64 = 1
	const toBlock int64 = 3
	// Prepare blocks in the DB
	blocks := setTestBlocks(fromBlock, toBlock)
	// Generate fake batches
	const nBatches = 9
	batches := test.GenBatches(nBatches, blocks)
	// Test GetLastL1TxsNum with no batches
	fetchedLastL1TxsNum, err := historyDB.GetLastL1TxsNum()
	assert.NoError(t, err)
	assert.Nil(t, fetchedLastL1TxsNum)
	// Add batches to the DB
	err = historyDB.AddBatches(batches)
	assert.NoError(t, err)
	// Get batches from the DB
	fetchedBatches, err := historyDB.GetBatches(0, common.BatchNum(nBatches))
	assert.NoError(t, err)
	for i, fetchedBatch := range fetchedBatches {
		assert.Equal(t, batches[i], fetchedBatch)
	}
	// Test GetLastBatchNum
	fetchedLastBatchNum, err := historyDB.GetLastBatchNum()
	assert.NoError(t, err)
	assert.Equal(t, batches[len(batches)-1].BatchNum, fetchedLastBatchNum)
	// Test GetLastL1TxsNum
	fetchedLastL1TxsNum, err = historyDB.GetLastL1TxsNum()
	assert.NoError(t, err)
	assert.Equal(t, *batches[nBatches-1].ForgeL1TxsNum, *fetchedLastL1TxsNum)
	// Test total fee
	// Generate fake tokens
	const nTokens = 5
	tokens, ethToken := test.GenTokens(nTokens, blocks)
	err = historyDB.AddTokens(tokens)
	assert.NoError(t, err)
	tokens = append([]common.Token{ethToken}, tokens...)
	feeBatch := batches[0]
	feeBatch.BatchNum = 9999
	feeBatch.CollectedFees = make(map[common.TokenID]*big.Int)
	var total float64
	for i, token := range tokens {
		value := 3.019237 * float64(i)
		assert.NoError(t, historyDB.UpdateTokenValue(token.Symbol, value))
		bigAmount := big.NewInt(345000000)
		feeBatch.CollectedFees[token.TokenID] = bigAmount
		f := new(big.Float).SetInt(bigAmount)
		amount, _ := f.Float64()
		total += value * (amount / math.Pow(10, float64(token.Decimals)))
	}
	err = historyDB.AddBatch(&feeBatch)
	assert.NoError(t, err)
	fetchedBatches, err = historyDB.GetBatches(feeBatch.BatchNum-1, feeBatch.BatchNum+1)
	assert.NoError(t, err)
	for _, fetchedBatch := range fetchedBatches {
		if fetchedBatch.BatchNum == feeBatch.BatchNum {
			assert.Equal(t, total, *fetchedBatch.TotalFeesUSD)
		}
	}
}

func TestBids(t *testing.T) {
	const fromBlock int64 = 1
	const toBlock int64 = 5
	// Prepare blocks in the DB
	blocks := setTestBlocks(fromBlock, toBlock)
	// Generate fake coordinators
	const nCoords = 5
	coords := test.GenCoordinators(nCoords, blocks)
	err := historyDB.AddCoordinators(coords)
	assert.NoError(t, err)
	// Generate fake bids
	const nBids = 20
	bids := test.GenBids(nBids, blocks, coords)
	err = historyDB.AddBids(bids)
	assert.NoError(t, err)
	// Fetch bids
	fetchedBids, err := historyDB.GetAllBids()
	assert.NoError(t, err)
	// Compare fetched bids vs generated bids
	for i, bid := range fetchedBids {
		assert.Equal(t, bids[i], bid)
	}
}

func TestTokens(t *testing.T) {
	const fromBlock int64 = 1
	const toBlock int64 = 5
	// Prepare blocks in the DB
	blocks := setTestBlocks(fromBlock, toBlock)
	// Generate fake tokens
	const nTokens = 5
	tokens, ethToken := test.GenTokens(nTokens, blocks)
	err := historyDB.AddTokens(tokens)
	assert.NoError(t, err)
	tokens = append([]common.Token{ethToken}, tokens...)
	limit := uint(10)
	// Fetch tokens6
	fetchedTokens, _, err := historyDB.GetTokens(nil, nil, "", nil, &limit, OrderAsc)
	assert.NoError(t, err)
	// Compare fetched tokens vs generated tokens
	// All the tokens should have USDUpdate setted by the DB trigger
	for i, token := range fetchedTokens {
		assert.Equal(t, tokens[i].TokenID, token.TokenID)
		assert.Equal(t, tokens[i].EthBlockNum, token.EthBlockNum)
		assert.Equal(t, tokens[i].EthAddr, token.EthAddr)
		assert.Equal(t, tokens[i].Name, token.Name)
		assert.Equal(t, tokens[i].Symbol, token.Symbol)
		assert.Nil(t, token.USD)
		assert.Nil(t, token.USDUpdate)
	}
}

func TestAccounts(t *testing.T) {
	const fromBlock int64 = 1
	const toBlock int64 = 5
	// Prepare blocks in the DB
	blocks := setTestBlocks(fromBlock, toBlock)
	// Generate fake tokens
	const nTokens = 5
	tokens, ethToken := test.GenTokens(nTokens, blocks)
	err := historyDB.AddTokens(tokens)
	assert.NoError(t, err)
	tokens = append([]common.Token{ethToken}, tokens...)
	// Generate fake batches
	const nBatches = 10
	batches := test.GenBatches(nBatches, blocks)
	err = historyDB.AddBatches(batches)
	assert.NoError(t, err)
	// Generate fake accounts
	const nAccounts = 3
	accs := test.GenAccounts(nAccounts, 0, tokens, nil, nil, batches)
	err = historyDB.AddAccounts(accs)
	assert.NoError(t, err)
	// Fetch accounts
	fetchedAccs, err := historyDB.GetAccounts()
	assert.NoError(t, err)
	// Compare fetched accounts vs generated accounts
	for i, acc := range fetchedAccs {
		accs[i].Balance = nil
		assert.Equal(t, accs[i], acc)
	}
}

func TestTxs(t *testing.T) {
	const fromBlock int64 = 1
	const toBlock int64 = 5
	// Prepare blocks in the DB
	blocks := setTestBlocks(fromBlock, toBlock)
	// Generate fake tokens
	const nTokens = 500
	tokens, ethToken := test.GenTokens(nTokens, blocks)
	err := historyDB.AddTokens(tokens)
	assert.NoError(t, err)
	tokens = append([]common.Token{ethToken}, tokens...)
	// Generate fake batches
	const nBatches = 10
	batches := test.GenBatches(nBatches, blocks)
	err = historyDB.AddBatches(batches)
	assert.NoError(t, err)
	// Generate fake accounts
	const nAccounts = 3
	accs := test.GenAccounts(nAccounts, 0, tokens, nil, nil, batches)
	err = historyDB.AddAccounts(accs)
	assert.NoError(t, err)

	/*
		Uncomment once the transaction generation is fixed
		!! test that batches that forge user L1s !!
		!! Missing tests to check that historic USD is not set if USDUpdate is too old (24h) !!

		// Generate fake L1 txs
		const nL1s = 64
		_, l1txs := test.GenL1Txs(256, nL1s, 0, nil, accs, tokens, blocks, batches)
		err = historyDB.AddL1Txs(l1txs)
		assert.NoError(t, err)
		// Generate fake L2 txs
		const nL2s = 2048 - nL1s
		_, l2txs := test.GenL2Txs(256, nL2s, 0, nil, accs, tokens, blocks, batches)
		err = historyDB.AddL2Txs(l2txs)
		assert.NoError(t, err)
		// Compare fetched txs vs generated txs.
		fetchAndAssertTxs(t, l1txs, l2txs)
		// Test trigger: L1 integrity
		// from_eth_addr can't be null
		l1txs[0].FromEthAddr = ethCommon.Address{}
		err = historyDB.AddL1Txs(l1txs)
		assert.Error(t, err)
		l1txs[0].FromEthAddr = ethCommon.BigToAddress(big.NewInt(int64(5)))
		// from_bjj can't be null
		l1txs[0].FromBJJ = nil
		err = historyDB.AddL1Txs(l1txs)
		assert.Error(t, err)
		privK := babyjub.NewRandPrivKey()
		l1txs[0].FromBJJ = privK.Public()
		// load_amount can't be null
		l1txs[0].LoadAmount = nil
		err = historyDB.AddL1Txs(l1txs)
		assert.Error(t, err)
		// Test trigger: L2 integrity
		// batch_num can't be null
		l2txs[0].BatchNum = 0
		err = historyDB.AddL2Txs(l2txs)
		assert.Error(t, err)
		l2txs[0].BatchNum = 1
		// nonce can't be null
		l2txs[0].Nonce = 0
		err = historyDB.AddL2Txs(l2txs)
		assert.Error(t, err)
		// Test trigger: forge L1 txs
		// add next batch to DB
		batchNum, toForgeL1TxsNum := test.GetNextToForgeNumAndBatch(batches)
		batch := batches[0]
		batch.BatchNum = batchNum
		batch.ForgeL1TxsNum = toForgeL1TxsNum
		assert.NoError(t, historyDB.AddBatch(&batch)) // This should update nL1s / 2 rows
		// Set batch num in txs that should have been marked as forged in the DB
		for i := 0; i < len(l1txs); i++ {
			fetchedTx, err := historyDB.GetTx(l1txs[i].TxID)
			assert.NoError(t, err)
			if l1txs[i].ToForgeL1TxsNum == toForgeL1TxsNum {
				assert.Equal(t, batchNum, *fetchedTx.BatchNum)
			} else {
				if fetchedTx.BatchNum != nil {
					assert.NotEqual(t, batchNum, *fetchedTx.BatchNum)
				}
			}
		}

		// Test helper functions for Synchronizer
		// GetLastTxsPosition
		expectedPosition := -1
		var choosenToForgeL1TxsNum int64 = -1
		for _, tx := range l1txs {
			if choosenToForgeL1TxsNum == -1 && tx.ToForgeL1TxsNum > 0 {
				choosenToForgeL1TxsNum = tx.ToForgeL1TxsNum
				expectedPosition = tx.Position
			} else if choosenToForgeL1TxsNum == tx.ToForgeL1TxsNum && expectedPosition < tx.Position {
				expectedPosition = tx.Position
			}
		}
		position, err := historyDB.GetLastTxsPosition(choosenToForgeL1TxsNum)
		assert.NoError(t, err)
		assert.Equal(t, expectedPosition, position)

		// GetL1UserTxs: not needed? tests were broken
		// txs, err := historyDB.GetL1UserTxs(2)
		// assert.NoError(t, err)
		// assert.NotZero(t, len(txs))
		// assert.NoError(t, err)
		// assert.Equal(t, 22, position)
		// // Test Update L1 TX Batch_num
		// assert.Equal(t, common.BatchNum(0), txs[0].BatchNum)
		// txs[0].BatchNum = common.BatchNum(1)
		// txs, err = historyDB.GetL1UserTxs(2)
		// assert.NoError(t, err)
		// assert.NotZero(t, len(txs))
		// assert.Equal(t, common.BatchNum(1), txs[0].BatchNum)
	*/
}

/*
func fetchAndAssertTxs(t *testing.T, l1txs []common.L1Tx, l2txs []common.L2Tx) {
	for i := 0; i < len(l1txs); i++ {
		tx := l1txs[i].Tx()
		fmt.Println("ASDF", i, tx.TxID)
		fetchedTx, err := historyDB.GetTx(tx.TxID)
		require.NoError(t, err)
		test.AssertUSD(t, tx.USD, fetchedTx.USD)
		test.AssertUSD(t, tx.LoadAmountUSD, fetchedTx.LoadAmountUSD)
		assert.Equal(t, tx, fetchedTx)
	}
	for i := 0; i < len(l2txs); i++ {
		tx := l2txs[i].Tx()
		fetchedTx, err := historyDB.GetTx(tx.TxID)
		tx.TokenID = fetchedTx.TokenID
		assert.NoError(t, err)
		test.AssertUSD(t, fetchedTx.USD, tx.USD)
		test.AssertUSD(t, fetchedTx.FeeUSD, tx.FeeUSD)
		assert.Equal(t, tx, fetchedTx)
	}
}
*/

func TestExitTree(t *testing.T) {
	nBatches := 17
	blocks := setTestBlocks(1, 10)
	batches := test.GenBatches(nBatches, blocks)
	err := historyDB.AddBatches(batches)
	assert.NoError(t, err)
	const nTokens = 50
	tokens, ethToken := test.GenTokens(nTokens, blocks)
	err = historyDB.AddTokens(tokens)
	assert.NoError(t, err)
	tokens = append([]common.Token{ethToken}, tokens...)
	const nAccounts = 3
	accs := test.GenAccounts(nAccounts, 0, tokens, nil, nil, batches)
	assert.NoError(t, historyDB.AddAccounts(accs))
	exitTree := test.GenExitTree(nBatches, batches, accs)
	err = historyDB.AddExitTree(exitTree)
	assert.NoError(t, err)
}

func TestGetL1UserTxs(t *testing.T) {
	test.WipeDB(historyDB.DB())

	set := `
		Type: Blockchain
		AddToken(1)
		AddToken(2)
		AddToken(3)

		CreateAccountDeposit(1) A: 20
		CreateAccountDeposit(2) A: 20
		CreateAccountDeposit(1) B: 5
		CreateAccountDeposit(1) C: 5
		CreateAccountDeposit(1) D: 5

		> block
	`
	tc := til.NewContext(128)
	blocks, err := tc.GenerateBlocks(set)
	require.Nil(t, err)
	// Sanity check
	require.Equal(t, 1, len(blocks))
	require.Equal(t, 5, len(blocks[0].L1UserTxs))
	// fmt.Printf("DBG Blocks: %+v\n", blocks)

	toForgeL1TxsNum := int64(1)

	for i := range blocks {
		err = historyDB.AddBlockSCData(&blocks[i])
		require.Nil(t, err)
	}

	l1UserTxs, err := historyDB.GetL1UserTxs(toForgeL1TxsNum)
	require.Nil(t, err)
	assert.Equal(t, 5, len(l1UserTxs))
	assert.Equal(t, blocks[0].L1UserTxs, l1UserTxs)

	// No l1UserTxs for this toForgeL1TxsNum
	l1UserTxs, err = historyDB.GetL1UserTxs(2)
	require.Nil(t, err)
	assert.Equal(t, 0, len(l1UserTxs))
}

// setTestBlocks WARNING: this will delete the blocks and recreate them
func setTestBlocks(from, to int64) []common.Block {
	test.WipeDB(historyDB.DB())
	blocks := test.GenBlocks(from, to)
	if err := historyDB.AddBlocks(blocks); err != nil {
		panic(err)
	}
	return blocks
}
