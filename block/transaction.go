package block

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"strings"
)

type Transaction struct {
	senderBlockchainAddress    string
	recipientBlockchainAddress string
	value                      int64
}

func (t *Transaction) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Sender    string `json:"sender_blockchain_address"`
		Recipient string `json:"recipient_blockchain_address"`
		Value     int64  `json:"value"`
	}{
		Sender:    t.senderBlockchainAddress,
		Recipient: t.recipientBlockchainAddress,
		Value:     t.value,
	})
}

func (t *Transaction) UnmarshalJSON(data []byte) error {
	v := &struct {
		Sender    *string `json:"sender_blockchain_address"`
		Recipient *string `json:"recipient_blockchain_address"`
		Value     *int64  `json:"value"`
	}{
		Sender:    &t.senderBlockchainAddress,
		Recipient: &t.recipientBlockchainAddress,
		Value:     &t.value,
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	return nil
}

func (t *Transaction) Hash() [32]byte {
	m, _ := json.Marshal(t)
	return sha256.Sum256([]byte(m))
}
func NewTransaction(sender string, recipient string, value int64) *Transaction {
	return &Transaction{sender, recipient, value}
}

func (t *Transaction) Print() {
	color.Red("%s\n", strings.Repeat("~", 30))
	color.Cyan("发送地址             %s\n", t.senderBlockchainAddress)
	color.Cyan("接受地址             %s\n", t.recipientBlockchainAddress)
	color.Cyan("金额                 %d\n", t.value)
}

type TransactionRequest struct {
	SenderBlockchainAddress    *string `json:"sender_blockchain_address"`
	RecipientBlockchainAddress *string `json:"recipient_blockchain_address"`
	SenderPublicKey            *string `json:"sender_public_key"`
	Value                      *uint64 `json:"value"`
	Signature                  *string `json:"signature"`
}

func (tr *TransactionRequest) Validate() bool {
	if tr.SenderBlockchainAddress == nil ||
		tr.RecipientBlockchainAddress == nil ||
		tr.SenderPublicKey == nil ||
		tr.Value == nil ||
		tr.Signature == nil {
		return false
	}
	return true
}

type TransactionResponse struct {
	txHash                     [32]byte
	senderBlockchainAddress    string
	recipientBlockchainAddress string
	value                      int64
	number                     int64
	timestamp                  int64
}

func (tp *TransactionResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		TxHash    string `json:"tx_hash"`
		Sender    string `json:"sender_blockchain_address"`
		Recipient string `json:"recipient_blockchain_address"`
		Value     int64  `json:"value"`
		Number    int64  `json:"number"`
		Timestamp int64  `json:"timestamp"`
	}{
		TxHash:    fmt.Sprintf("%x", tp.txHash),
		Sender:    tp.senderBlockchainAddress,
		Recipient: tp.recipientBlockchainAddress,
		Value:     tp.value,
		Number:    tp.number,
		Timestamp: tp.timestamp,
	})
}
func (tp *TransactionResponse) UnmarshalJSON(data []byte) error {
	var txHash string
	temp := &struct {
		TxHash    *string `json:"tx_hash"`
		Sender    *string `json:"sender_blockchain_address"`
		Recipient *string `json:"recipient_blockchain_address"`
		Value     *int64  `json:"value"`
		Number    *int64  `json:"number"`
		Timestamp *int64  `json:"timestamp"`
	}{
		TxHash:    &txHash,
		Sender:    &tp.senderBlockchainAddress,
		Recipient: &tp.recipientBlockchainAddress,
		Value:     &tp.value,
		Number:    &tp.number,
		Timestamp: &tp.timestamp,
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	ph, _ := hex.DecodeString(*temp.TxHash)
	copy(tp.txHash[:], ph[:32])
	return nil
}
