package block

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

type Block struct {
	nonce        int64
	number       int64
	difficulty   int64
	timestamp    int64
	previousHash [32]byte
	transactions []*Transaction
}

func (b *Block) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Timestamp    int64          `json:"timestamp"`
		Nonce        int64          `json:"nonce"`
		Number       int64          `json:"number"`
		Difficulty   int64          `json:"difficulty"`
		PreviousHash string         `json:"previous_hash"`
		Transactions []*Transaction `json:"transactions"`
	}{
		Timestamp:    b.timestamp,
		Nonce:        b.nonce,
		Number:       b.number,
		Difficulty:   b.difficulty,
		PreviousHash: fmt.Sprintf("%x", b.previousHash),
		Transactions: b.transactions,
	})
}

func (b *Block) UnmarshalJSON(data []byte) error {
	var previousHash string
	v := &struct {
		Timestamp    *int64          `json:"timestamp"`
		Nonce        *int64          `json:"nonce"`
		Number       *int64          `json:"number"`
		Difficulty   *int64          `json:"difficulty"`
		PreviousHash *string         `json:"previous_hash"`
		Transactions *[]*Transaction `json:"transactions"`
	}{
		Timestamp:    &b.timestamp,
		Nonce:        &b.nonce,
		Number:       &b.number,
		Difficulty:   &b.difficulty,
		PreviousHash: &previousHash,
		Transactions: &b.transactions,
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	ph, _ := hex.DecodeString(*v.PreviousHash)
	copy(b.previousHash[:], ph[:32])
	return nil
}

func (b *Block) Hash() [32]byte {
	m, _ := json.Marshal(b)
	return sha256.Sum256([]byte(m))
}

func (b *Block) PreviousHash() [32]byte {
	return b.previousHash
}
func (b *Block) Number() int64 {
	return b.number
}
func (b *Block) Nonce() int64 {
	return b.nonce
}
func (b *Block) Difficulty() int {
	return int(b.difficulty)
}

func (b *Block) Transactions() []*Transaction {
	return b.transactions
}
func NewBlock(nonce int64, difficulty int64, number int64, previousHash [32]byte, transactions []*Transaction) *Block {
	b := new(Block)
	b.timestamp = time.Now().UnixNano()
	b.nonce = nonce
	b.difficulty = difficulty
	b.number = number
	b.previousHash = previousHash
	b.transactions = transactions
	return b
}

func (b *Block) Print() {
	fmt.Printf("%-15v:%30d\n", "Block Number", b.number)
	fmt.Printf("%-15v:%30d\n", "Nonce", b.nonce)
	fmt.Printf("%-15v:%30d\n", "Difficulty", b.difficulty)
	fmt.Printf("%-15v:%30d\n", "Timestamp", b.timestamp)
	fmt.Printf("%-15v:%30x\n", "Previous Hash", b.previousHash)
	for _, t := range b.transactions {
		t.Print()
	}
}
