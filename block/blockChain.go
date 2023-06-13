package block

import (
	"KaiFuBlockChain/utils"
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

var MINING_DIFFICULT = 0x80000

const (
	COIN_POOL_Bqno         = "BqnoiMm4Pot5EDGjV3yR7rKXvuTaQbn1qn8zkykJnjQ9" //私钥6666
	MINING_ACCOUNT_ADDRESS = "KAIFU BLOCKCHAIN"
	MINING_REWARD          = 500
	MINING_TIMER_SEC       = 10 //自动挖矿间隔 十秒钟

	BLOCKCHAIN_PORT_RANGE_START      = 5000
	BLOCKCHAIN_PORT_RANGE_END        = 5002
	NEIGHBOR_IP_RANGE_START          = 0
	NEIGHBOR_IP_RANGE_END            = 0
	BLOCKCHIN_NEIGHBOR_SYNC_TIME_SEC = 10 //同步邻居时间间隔 十秒钟
)

type Blockchain struct {
	transactionPool   []*Transaction
	chain             []*Block
	blockchainAddress string
	port              uint16
	mux               sync.Mutex
	neighbors         []string
	muxNeighbors      sync.Mutex
}

func (bc *Blockchain) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Blocks            []*Block `json:"chain"`
		BlockchainAddress string   `json:"blockchainAddress"`
		Port              uint16   `json:"port"`
	}{
		Blocks:            bc.chain,
		BlockchainAddress: bc.blockchainAddress,
		Port:              bc.port,
	})
}

func (bc *Blockchain) UnmarshalJSON(data []byte) error {
	v := &struct {
		Blocks            *[]*Block `json:"chain"`
		BlockchainAddress *string   `json:"blockchainAddress"`
		Port              *uint16   `json:"port"`
	}{
		Blocks:            &bc.chain,
		BlockchainAddress: &bc.blockchainAddress,
		Port:              &bc.port,
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	return nil
}

func (bc *Blockchain) TransactionPool() []*Transaction {
	return bc.transactionPool
}
func (bc *Blockchain) ClearTransactionPool() {
	bc.transactionPool = bc.transactionPool[:0]
}

func (bc *Blockchain) BlockchainAddress() string {
	return bc.blockchainAddress
}
func (bc *Blockchain) Port() uint16 {
	return bc.port
}
func (bc *Blockchain) Chain() []*Block {
	return bc.chain
}

// GetBlockByNumber 区块链号查询区块
func (bc *Blockchain) GetBlockByNumber(number int64) *Block {
	for _, block := range bc.chain {
		if block.number == number {
			return block
		}
	}
	return nil
}

// GetBlockByHash 哈希查询区块
func (bc *Blockchain) GetBlockByHash(hash [32]byte) *Block {
	for _, block := range bc.chain {
		if block.Hash() == hash {
			return block
		}
	}
	return nil
}

// GetTransactionByHash 哈希查询交易（包括链上和交易池）
func (bc *Blockchain) GetTransactionByHash(hash [32]byte) *Transaction {
	//查链上  Completed
	for _, block := range bc.chain {
		for _, tx := range block.transactions {
			if tx.Hash() == hash {
				return tx
			}
		}
	}

	//查交易池 Pending
	for _, tx := range bc.transactionPool {
		if tx.Hash() == hash {
			return tx
		}
	}

	return nil
}

// GetTransactionsByAddress 某地址的交易历史

func (bc *Blockchain) GetTransactionsByAddress(blockchainAddress string) []*TransactionResponse {

	var completed []*TransactionResponse
	// 在区块链中查找
	for _, block := range bc.chain {
		for _, tx := range block.transactions {
			if tx.senderBlockchainAddress == blockchainAddress || tx.recipientBlockchainAddress == blockchainAddress {
				tmpTp := &TransactionResponse{
					txHash:                     tx.Hash(),
					senderBlockchainAddress:    tx.senderBlockchainAddress,
					recipientBlockchainAddress: tx.recipientBlockchainAddress,
					value:                      tx.value, number: block.Number(),
					timestamp: block.timestamp,
				}
				completed = append(completed, tmpTp)
			}
		}
	}

	return completed
}

func (bc *Blockchain) LastBlock() *Block {
	if len(bc.chain) >= 1 {
		return bc.chain[len(bc.chain)-1]
	}
	return nil
}

func (bc *Blockchain) Print() {
	for i, block := range bc.chain {
		color.Green("%s BLOCK %d %s\n", strings.Repeat("=", 25), i, strings.Repeat("=", 25))
		block.Print()
	}
	color.Yellow("%s\n\n\n", strings.Repeat("*", 50))
}

func (bc *Blockchain) Run() {
	bc.StartSyncNeighbors()
	bc.ResolveConflicts()
	bc.StartMining()
}

// -----------------BlockChain Begin--------------------------

// CreateBlock 挖矿成功后向区块链上追加区块
func (bc *Blockchain) CreateBlock(nonce int64, difficulty int64, previousHash [32]byte) *Block {

	var number int64 = 0
	if bc.LastBlock() != nil {
		number = bc.LastBlock().Number() + 1
	}
	b := NewBlock(nonce, difficulty, number, previousHash, bc.transactionPool)
	bc.chain = append(bc.chain, b)
	bc.transactionPool = []*Transaction{}

	//宣告我是出块者，让其他节点清空自己的交易池停止挖矿
	for _, n := range bc.neighbors {
		endpoint := fmt.Sprintf("http://%s/transactions", n)
		client := &http.Client{}
		//DELETE为广播清空交易池
		req, _ := http.NewRequest("DELETE", endpoint, nil)
		resp, _ := client.Do(req)
		body, _ := ioutil.ReadAll(resp.Body)
		log.Printf(string(body))
	}
	return b
}

// NewBlockchain 新建一条链的第一个区块
func NewBlockchain(blockchainAddress string, port uint16) *Blockchain {
	b := &Block{}
	bc := new(Blockchain)
	bc.blockchainAddress = blockchainAddress
	bc.CreateBlock(0, int64(MINING_DIFFICULT), b.Hash())
	bc.port = port
	return bc
}

// -----------------Multi Neighbors Communication--------------

func (bc *Blockchain) ValidChain(chain []*Block) bool {
	preBlock := chain[0]
	currentIndex := 1
	for currentIndex < len(chain) {
		b := chain[currentIndex]
		if b.previousHash != preBlock.Hash() {
			return false
		}
		preBlock = b
		currentIndex += 1
	}
	return true
}

func (bc *Blockchain) ResolveConflicts() bool {
	var longestChain []*Block = nil
	maxLength := len(bc.chain)
	fmt.Println("本链长度：", maxLength)
	for _, n := range bc.neighbors {
		endpoint := fmt.Sprintf("http://%s/chain", n) //请求邻居节点的区块链
		resp, _ := http.Get(endpoint)
		if resp.StatusCode == 200 {
			var bcResp Blockchain
			decoder := json.NewDecoder(resp.Body)
			_ = decoder.Decode(&bcResp)

			chain := bcResp.Chain()
			if len(chain) > maxLength && bc.ValidChain(chain) {
				maxLength = len(chain)
				longestChain = chain
			}
		}
	}

	if longestChain != nil {
		fmt.Println("最新链长度", len(longestChain))
		bc.chain = longestChain //更新为最长链
		color.HiGreen("[OK] 最新分叉已修正")
		return true
	}
	color.HiYellow("[WARN] 注意，当前链上数据可能未更新")
	return false
}

func (bc *Blockchain) SetNeighbors() {
	bc.neighbors = utils.FindNeighbors(
		utils.GetHost(), bc.port,
		NEIGHBOR_IP_RANGE_START, NEIGHBOR_IP_RANGE_END,
		BLOCKCHAIN_PORT_RANGE_START, BLOCKCHAIN_PORT_RANGE_END)
	color.Blue("邻居节点：%v", bc.neighbors)
}

func (bc *Blockchain) SyncNeighbors() {
	bc.muxNeighbors.Lock()
	defer bc.muxNeighbors.Unlock()
	bc.SetNeighbors()
}

func (bc *Blockchain) StartSyncNeighbors() {
	bc.SyncNeighbors()
	_ = time.AfterFunc(time.Second*BLOCKCHIN_NEIGHBOR_SYNC_TIME_SEC, bc.StartSyncNeighbors)
}

// -----------------Transaction-----------------

// CalculateTotalAmount 计算余额
func (bc *Blockchain) CalculateTotalAmount(accountAddress string) uint64 {
	var totalAmount uint64 = 0
	for _, _chain := range bc.chain {
		for _, _tx := range _chain.transactions {
			if accountAddress == _tx.recipientBlockchainAddress {
				totalAmount = totalAmount + uint64(_tx.value)
			}
			if accountAddress == _tx.senderBlockchainAddress {
				totalAmount = totalAmount - uint64(_tx.value)
			}
		}
	}
	//如果有转出交易还在交易池里还未打包，应该也要暂时扣除余额里，如果打包失败金额就会退回
	for _, _tx := range bc.TransactionPool() {
		if accountAddress == _tx.senderBlockchainAddress {
			totalAmount = totalAmount - uint64(_tx.value)
		}
	}
	return totalAmount
}

// VerifyTransactionSignature 验证交易签名
func (bc *Blockchain) VerifyTransactionSignature(
	senderPublicKey *ecdsa.PublicKey, s *utils.Signature, t *Transaction) bool {
	m, _ := json.Marshal(t)
	h := sha256.Sum256([]byte(m))
	return ecdsa.Verify(senderPublicKey, h[:], s.R, s.S)
}

// AddTransaction 添加交易到交易池
func (bc *Blockchain) AddTransaction(sender string, recipient string, value int64,
	senderPublicKey *ecdsa.PublicKey, s *utils.Signature) bool {
	t := NewTransaction(sender, recipient, value)

	//如果是挖矿得到的奖励交易，不验证
	if sender == MINING_ACCOUNT_ADDRESS {
		bc.transactionPool = append(bc.transactionPool, t)
		return true
	}

	//如果是币池发币交易，不验证
	if sender == COIN_POOL_Bqno {
		bc.transactionPool = append(bc.transactionPool, t)
		return true
	}

	if bc.VerifyTransactionSignature(senderPublicKey, s, t) {

		if value <= 0 {
			color.HiRed("[ERROR]: 转账金额必须大于0！")
			return false
		}

		if bc.CalculateTotalAmount(sender) < uint64(value) {
			color.HiRed("[ERROR]: 余额不足")
			return false
		}

		bc.transactionPool = append(bc.transactionPool, t)
		return true
	} else {
		color.HiRed("[ERROR]: 交易验证失败")
	}
	return false

}

// CreateTransaction 创建交易并广播给邻居节点
func (bc *Blockchain) CreateTransaction(sender string, recipient string, value uint64,
	senderPublicKey *ecdsa.PublicKey, s *utils.Signature) bool {
	isTransacted := bc.AddTransaction(sender, recipient, int64(value), senderPublicKey, s)
	fmt.Println("开始准备广播")
	if isTransacted {
		fmt.Println("正在广播...")
		//开始广播，让邻居们也赶紧AddTransaction
		for _, n := range bc.neighbors {

			publicKeyStr := fmt.Sprintf("%064x%064x", senderPublicKey.X.Bytes(),
				senderPublicKey.Y.Bytes())
			signatureStr := s.String()
			bt := &TransactionRequest{
				&sender, &recipient, &publicKeyStr, &value, &signatureStr}
			m, _ := json.Marshal(bt)
			buf := bytes.NewBuffer(m)
			endpoint := fmt.Sprintf("http://%s/transactions", n)
			client := &http.Client{}
			req, _ := http.NewRequest("PUT", endpoint, buf) //注意是PUT
			resp, _ := client.Do(req)
			body, _ := ioutil.ReadAll(resp.Body)
			log.Printf(string(body))
		}
	}

	return isTransacted
}

type AmountResponse struct {
	Amount uint64 `json:"amount"`
}

func (ar *AmountResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Amount uint64 `json:"amount"`
	}{
		Amount: ar.Amount,
	})
}

//-----------------Mining-----------------

// CopyTransactionPool 拷贝交易池用于PoW
func (bc *Blockchain) CopyTransactionPool() []*Transaction {
	transactions := make([]*Transaction, 0)
	for _, t := range bc.transactionPool {
		transactions = append(transactions,
			NewTransaction(t.senderBlockchainAddress,
				t.recipientBlockchainAddress,
				t.value))
	}
	return transactions
}

func bytesToBigInt(b [32]byte) *big.Int {
	// 转换为 []byte 类型
	bytes := b[:]
	// 调用 SetBytes() 函数进行转换
	result := new(big.Int).SetBytes(bytes)
	return result
}

// ValidProof 区块有效性验证
func (bc *Blockchain) ValidProof(nonce int64, previousHash [32]byte, transactions []*Transaction, difficulty int) bool {
	bigi_2 := big.NewInt(2)
	bigi_256 := big.NewInt(256)
	bigi_diff := big.NewInt(int64(difficulty))

	target := new(big.Int).Exp(bigi_2, bigi_256, nil) //计算2^256的结果
	target = new(big.Int).Div(target, bigi_diff)      //计算的2^256结果除以difficulty，得到挖矿的目标值

	tmpBlock := Block{nonce: nonce, previousHash: previousHash, transactions: transactions}
	result := bytesToBigInt(tmpBlock.Hash())

	return target.Cmp(result) > 0 //目标值大于实际值，则返回true，表示挖矿有效
}

// Avg 平均出块时间
func (bc *Blockchain) Avg() int64 {
	length := len(bc.chain)
	return (bc.chain[length-1].timestamp - bc.chain[0].timestamp) / int64(length)
}

// getBlockSpendTime 第num个区块的出块时间
func (bc *Blockchain) getBlockSpendTime(num int) int64 {
	if num == 0 {
		return 0
	}
	return bc.chain[num].timestamp - bc.chain[num-1].timestamp
}

// ProofOfWork 工作量证明计算nonce值
func (bc *Blockchain) ProofOfWork() int64 {
	// 1，得到最近100个区块的平均时间
	// 2如果小于5秒，MINING_DIFFICULT增加4.5
	// 3判断一个难度上限

	transactions := bc.CopyTransactionPool()
	previousHash := bc.LastBlock().Hash()
	var nonce int64 = 0
	begin := time.Now()

	// 根据上一个区块的时间，来判断是不是在目标区间来调整 MINING_DIFFICULT

	if bc.getBlockSpendTime(len(bc.chain)-1) < 3e+9 {
		// MINING_DIFFICULT++   //原
		MINING_DIFFICULT += 32
	} else {
		if MINING_DIFFICULT >= 130000 {
			// MINING_DIFFICULT-- //原
			MINING_DIFFICULT -= 32
		}
	}

	for !bc.ValidProof(nonce, previousHash, transactions, MINING_DIFFICULT) {
		nonce += 1
	}
	end := time.Now()

	log.Printf("Block %d POW spend Time:%f Second, Diff:%d", len(bc.chain)+1, end.Sub(begin).Seconds(), MINING_DIFFICULT)
	// log.Printf("POW spend Time:%s", end.Sub(begin))

	return nonce
}

// Mining 挖矿
func (bc *Blockchain) Mining() bool {
	bc.mux.Lock()
	defer bc.mux.Unlock()

	// 无交易区块不打包
	// 在这里判断可能会遗漏让两个正在节点已经走到下面的PoW函数里，
	// 但是率先挖出的人会清空对方交易池导致无交易区块被打包
	if len(bc.transactionPool) == 0 {
		return false
	}

	bc.AddTransaction(MINING_ACCOUNT_ADDRESS, bc.blockchainAddress, MINING_REWARD, nil, nil)
	nonce := bc.ProofOfWork()
	previousHash := bc.LastBlock().Hash()

	//所以这里再做一个if 这个if是防止打包空交易块
	if len(bc.transactionPool) == 0 {

		color.HiWhite("这个块的交易池已经被提前挖出的节点清空了")
		color.HiWhite("本案例为了保证局域网区块链一致性不处理软分叉，所以本次挖矿不算成功，不添加到区块链")
		return false
	}

	//注意 这里的挖矿成功后的CreateBlock要广播邻居节点清空交易池，宣布这个区块由我率先挖出
	bc.CreateBlock(nonce, int64(MINING_DIFFICULT), previousHash)
	log.Println("action=mining, status=success")

	//谁先挖出来这个矿就让大家去共识一下
	for _, n := range bc.neighbors {
		endpoint := fmt.Sprintf("http://%s/consensus", n)
		client := &http.Client{}
		req, _ := http.NewRequest("PUT", endpoint, nil)
		resp, _ := client.Do(req)
		body, _ := ioutil.ReadAll(resp.Body)
		log.Printf(string(body))
	}

	return true
}

// 定义一个全局标志位来控制挖矿的状态
var miningStopped bool

// AutoMining 递归定义自动定时挖矿
func (bc *Blockchain) AutoMining() {
	if miningStopped {
		color.Yellow("time: %v自动挖矿已停止", time.Now().Format("2006-01-02 15:04:05"))
		return
	}
	isPackaged := bc.Mining()
	if isPackaged {
		color.HiYellow("本次挖矿打包成功!🔨minetime: %v\n", time.Now().Format("2006-01-02 15:04:05"))
	} else {
		color.Yellow("time:%v交易池为空本次挖矿暂停  ", time.Now().Format("2006-01-02 15:04:05"))
	}
	_ = time.AfterFunc(time.Second*MINING_TIMER_SEC, bc.AutoMining)
}

// StartMining 停止自动定时挖矿
func (bc *Blockchain) StartMining() {
	miningStopped = false
	color.Yellow("自动挖矿开始，每%d秒执行一次挖矿程序", MINING_TIMER_SEC)
	bc.AutoMining()
}

// StopMining 停止自动定时挖矿
func (bc *Blockchain) StopMining() {
	miningStopped = true
}
