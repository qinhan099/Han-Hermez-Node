package debugapi

import (
	"crypto/ecdsa"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"strconv"
	"testing"

	"github.com/dghubble/sling"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/hermeznetwork/hermez-node/common"
	"github.com/hermeznetwork/hermez-node/db/statedb"
	"github.com/iden3/go-iden3-crypto/babyjub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func newAccount(t *testing.T, i int) *common.Account {
	var sk babyjub.PrivateKey
	copy(sk[:], []byte(strconv.Itoa(i))) // only for testing
	pk := sk.Public()

	var key ecdsa.PrivateKey
	key.D = big.NewInt(int64(i + 1)) // only for testing
	key.PublicKey.X, key.PublicKey.Y = ethCrypto.S256().ScalarBaseMult(key.D.Bytes())
	key.Curve = ethCrypto.S256()
	address := ethCrypto.PubkeyToAddress(key.PublicKey)

	return &common.Account{
		Idx:       common.Idx(256 + i),
		TokenID:   common.TokenID(i),
		Nonce:     common.Nonce(i),
		Balance:   big.NewInt(1000),
		PublicKey: pk,
		EthAddr:   address,
	}
}

func TestDebugAPI(t *testing.T) {
	dir, err := ioutil.TempDir("", "tmpdb")
	require.Nil(t, err)

	sdb, err := statedb.NewStateDB(dir, statedb.TypeSynchronizer, 32)
	require.Nil(t, err)
	err = sdb.MakeCheckpoint() // Make a checkpoint to increment the batchNum
	require.Nil(t, err)

	addr := "localhost:12345"
	debugAPI := NewDebugAPI(addr, sdb)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		err := debugAPI.Run(ctx)
		require.Nil(t, err)
	}()

	var accounts []common.Account
	for i := 0; i < 16; i++ {
		account := newAccount(t, i)
		accounts = append(accounts, *account)
		_, err = sdb.CreateAccount(account.Idx, account)
		require.Nil(t, err)
	}

	url := fmt.Sprintf("http://%v/debug/", addr)

	var batchNum common.BatchNum
	req, err := sling.New().Get(url).Path("sdb/batchnum").ReceiveSuccess(&batchNum)
	require.Equal(t, http.StatusOK, req.StatusCode)
	require.Nil(t, err)
	assert.Equal(t, common.BatchNum(1), batchNum)

	var mtroot *big.Int
	req, err = sling.New().Get(url).Path("sdb/mtroot").ReceiveSuccess(&mtroot)
	require.Equal(t, http.StatusOK, req.StatusCode)
	require.Nil(t, err)
	// Testing against a hardcoded value obtained by running the test and
	// printing the value previously.
	assert.Equal(t, "8902613552504893273500019895709436294962812188236308621387152512232191202510",
		mtroot.String())

	var accountAPI common.Account
	req, err = sling.New().Get(url).
		Path(fmt.Sprintf("sdb/accounts/%v", accounts[0].Idx)).
		ReceiveSuccess(&accountAPI)
	require.Equal(t, http.StatusOK, req.StatusCode)
	require.Nil(t, err)
	assert.Equal(t, accounts[0], accountAPI)

	var accountsAPI []common.Account
	req, err = sling.New().Get(url).Path("sdb/accounts").ReceiveSuccess(&accountsAPI)
	require.Equal(t, http.StatusOK, req.StatusCode)
	require.Nil(t, err)
	assert.Equal(t, accounts, accountsAPI)

	cancel()
}