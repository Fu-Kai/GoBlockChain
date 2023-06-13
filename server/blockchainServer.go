package main

import (
	"KaiFuBlockChain/block"
	"KaiFuBlockChain/utils"
	"KaiFuBlockChain/wallet"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
)

// BlockchainServer 节点服务器
type BlockchainServer struct {
	port            uint16
	filePath        string
	cacheBlockChain *block.Blockchain //用于存放本地文件的缓存
}

// NewBlockchainServer 创建一个端口为port的节点服务器实例
func NewBlockchainServer(port uint16) *BlockchainServer {
	return &BlockchainServer{
		port:     port,
		filePath: strconv.Itoa(int(port)) + ".json",
	}
}

func (bcs *BlockchainServer) Port() uint16 {
	return bcs.port
}

// GetBlockChain 读取内存中的区块链
func (bcs *BlockchainServer) GetBlockChain() *block.Blockchain {
	return bcs.cacheBlockChain
}

// UpdateCache 更新内存中的区块链
func (bcs *BlockchainServer) UpdateCache(bc *block.Blockchain) {
	bcs.cacheBlockChain = bc
}

// SetBlockchainFromFile 把本地文件读取到内存中来
func (bcs *BlockchainServer) SetBlockchainFromFile() bool {

	// 如果本地文件不存在，就直接新建链到内存中，之后再创建一个本地文件
	if _, err := os.Stat(bcs.filePath); os.IsNotExist(err) {
		color.HiRed("未能加载区块链本地文件%s", err)
		minersWallet := wallet.NewWallet()

		// 地址和端口为了区别不同的节点
		bc := block.NewBlockchain(minersWallet.BlockchainAddress(), bcs.Port())
		bcs.UpdateCache(bc)
		color.Magenta("======新建的矿工帐号信息======")
		color.Magenta("矿工private_key(妥善保管)")
		color.HiMagenta("%v \n", minersWallet.PrivateKeyStr())
		color.Magenta("矿工public_key\n %v\n", minersWallet.PublicKeyStr())
		color.Magenta("矿工blockchain_address\n %s\n", minersWallet.BlockchainAddress())
		color.Magenta("============================\n")
	} else {
		// 如果本地文件存在，内存cache就从本地加载
		cache, err := loadCacheFromFile(bcs.filePath)
		fmt.Println("loadPort", cache.Port())
		if err != nil {
			log.Fatalf("文件读取失败: %v", err)
		}
		bcs.UpdateCache(cache)
		fmt.Println("文件加载到内存成功")
		color.Magenta("===从本地加载的区块链====\n")
		color.Magenta("Port\n")
		color.HiMagenta("%d\n", bcs.GetBlockChain().Port())
		color.Magenta("矿工blockchain_address\n")
		color.HiMagenta("%s\n", bcs.GetBlockChain().BlockchainAddress())
		color.Magenta("=========================\n")
	}
	return true
}

//-------------------http func------------------

func (bcs *BlockchainServer) GetChain(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		w.Header().Add("Content-Type", "application/json")
		bc := bcs.GetBlockChain()
		m, _ := bc.MarshalJSON()
		io.WriteString(w, string(m[:]))
	default:
		log.Printf("ERROR: Invalid HTTP Method")
	}
}

func (bcs *BlockchainServer) Transactions(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		{
			// Get:显示交易池的内容，Mine成功后清空交易池
			w.Header().Add("Content-Type", "application/json")
			bc := bcs.GetBlockChain()

			transactions := bc.TransactionPool()
			m, _ := json.Marshal(struct {
				Transactions []*block.Transaction `json:"transactions"`
				Length       int                  `json:"length"`
			}{
				Transactions: transactions,
				Length:       len(transactions),
			})
			io.WriteString(w, string(m[:]))
		}
	case http.MethodPost:
		{
			//Post: 接收钱包端的交易请求
			log.Printf("\n\n\n")
			log.Println("接受到wallet发送的交易")
			decoder := json.NewDecoder(req.Body)
			var t block.TransactionRequest
			err := decoder.Decode(&t)
			if err != nil {
				log.Printf("ERROR: %v", err)
				io.WriteString(w, string(utils.JsonStatus("Decode Transaction失败")))
				return
			}

			log.Println("发送人公钥SenderPublicKey:", *t.SenderPublicKey)
			log.Println("发送人私钥SenderPrivateKey:", *t.SenderBlockchainAddress)
			log.Println("接收人地址RecipientBlockchainAddress:", *t.RecipientBlockchainAddress)
			log.Println("金额Value:", *t.Value)
			log.Println("交易Signature:", *t.Signature)

			if !t.Validate() {
				log.Println("ERROR: missing field(s)")
				io.WriteString(w, string(utils.JsonStatus("fail")))
				return
			}

			publicKey := utils.PublicKeyFromString(*t.SenderPublicKey)
			signature := utils.SignatureFromString(*t.Signature)
			bc := bcs.GetBlockChain()

			isCreated := bc.CreateTransaction(*t.SenderBlockchainAddress,
				*t.RecipientBlockchainAddress, uint64(*t.Value), publicKey, signature)

			w.Header().Add("Content-Type", "application/json")
			var m []byte
			if !isCreated {
				w.WriteHeader(http.StatusBadRequest)
				m = utils.JsonStatus("fail[from:blockchainServer]")
			} else {
				w.WriteHeader(http.StatusCreated)
				m = utils.JsonStatus("success[from:blockchainServer]")
			}
			io.WriteString(w, string(m))

		}
	case http.MethodPut:
		//Put: 接收其他节点server端的交易广播并AddTransaction到交易池
		decoder := json.NewDecoder(req.Body)
		var t block.TransactionRequest
		err := decoder.Decode(&t)
		if err != nil {
			log.Printf("ERROR: %v", err)
			io.WriteString(w, string(utils.JsonStatus("fail")))
			return
		}
		if !t.Validate() {
			log.Println("ERROR: missing field(s)")
			io.WriteString(w, string(utils.JsonStatus("fail")))
			return
		}
		color.HiWhite("已收到来自其他节点的交易广播")
		publicKey := utils.PublicKeyFromString(*t.SenderPublicKey)
		signature := utils.SignatureFromString(*t.Signature)
		bc := bcs.GetBlockChain()
		isAdded := bc.AddTransaction(*t.SenderBlockchainAddress,
			*t.RecipientBlockchainAddress, int64(*t.Value), publicKey, signature)

		w.Header().Add("Content-Type", "application/json")
		var m []byte
		if !isAdded {
			w.WriteHeader(http.StatusBadRequest)
			m = utils.JsonStatus("fail")
		} else {
			m = utils.JsonStatus("success 被广播的" + strconv.Itoa(int(bcs.Port())) + "邻居节点已将交易添加至交易池")
		}
		io.WriteString(w, string(m))
	case http.MethodDelete:
		//DELETE: 接收其他节点server端的打包完成信号，清空自己交易池
		bc := bcs.GetBlockChain()
		bc.ClearTransactionPool()
		color.HiWhite("已有节点率先打包完成，本节点被清空交易池")
		io.WriteString(w, string(utils.JsonStatus("success 被通知的"+strconv.Itoa(int(bcs.Port()))+"邻居节点已清空交易池")))
	default:
		log.Println("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) Mine(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		bc := bcs.GetBlockChain()
		isMined := bc.Mining()

		var m []byte
		if !isMined {
			w.WriteHeader(http.StatusBadRequest)
			m = utils.JsonStatus("挖矿失败[from:Mine]")
		} else {
			m = utils.JsonStatus("挖矿成功[from:Mine]")
		}
		w.Header().Add("Content-Type", "application/json")
		io.WriteString(w, string(m))
	default:
		log.Println("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) StartMine(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		bc := bcs.GetBlockChain()
		bc.StartMining()

		m := utils.JsonStatus("success")
		w.Header().Add("Content-Type", "application/json")
		io.WriteString(w, string(m))
	default:
		log.Println("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) StopMine(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		bc := bcs.GetBlockChain()
		bc.StopMining()

		m := utils.JsonStatus("success")
		w.Header().Add("Content-Type", "application/json")
		io.WriteString(w, string(m))
	default:
		log.Println("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) History(w http.ResponseWriter, req *http.Request) {

	switch req.Method {
	case http.MethodPost:
		w.Header().Add("Content-Type", "application/json")
		var data map[string]interface{}
		// 解析JSON数据

		err := json.NewDecoder(req.Body).Decode(&data)
		if err != nil {
			http.Error(w, "无法解析JSON数据", http.StatusBadRequest)
			return
		}
		// 获取JSON字段的值
		blockchainAddress := data["blockchain_address"].(string)

		color.Green("查询账户: %s 历史交易请求", blockchainAddress)

		completed := bcs.GetBlockChain().GetTransactionsByAddress(blockchainAddress)

		m, _ := json.Marshal(struct {
			CompletedTransactions []*block.TransactionResponse `json:"completedTransactions"`
			Length                int                          `json:"length"`
		}{
			CompletedTransactions: completed,
			Length:                len(completed),
		})
		io.WriteString(w, string(m[:]))

	default:
		log.Printf("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) Amount(w http.ResponseWriter, req *http.Request) {

	switch req.Method {
	case http.MethodPost:

		var data map[string]interface{}
		// 解析JSON数据

		err := json.NewDecoder(req.Body).Decode(&data)
		if err != nil {
			http.Error(w, "无法解析JSON数据", http.StatusBadRequest)
			return
		}
		// 获取JSON字段的值
		blockchainAddress := data["blockchain_address"].(string)

		color.Green("查询账户: %s 余额请求", blockchainAddress)

		amount := bcs.GetBlockChain().CalculateTotalAmount(blockchainAddress)

		ar := &block.AmountResponse{Amount: amount}
		m, _ := ar.MarshalJSON()

		w.Header().Add("Content-Type", "application/json")
		io.WriteString(w, string(m[:]))

	default:
		log.Printf("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}
func (bcs *BlockchainServer) Consensus(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPut:
		color.Cyan("####################Consensus###############")
		bc := bcs.GetBlockChain()
		replaced := bc.ResolveConflicts()
		color.Red("[共识]Consensus replaced :%v\n", replaced)
		w.Header().Add("Content-Type", "application/json")
		if replaced {
			io.WriteString(w, string(utils.JsonStatus("success 被通知的"+strconv.Itoa(int(bcs.Port()))+"邻居节点已经完成共识")))
		} else {
			io.WriteString(w, string(utils.JsonStatus("fail")))
		}
	default:
		log.Printf("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) GetBlockByNumber(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:

		var data map[string]interface{}
		// 解析JSON数据

		err := json.NewDecoder(req.Body).Decode(&data)
		if err != nil {
			http.Error(w, "无法解析JSON数据", http.StatusBadRequest)
			return
		}
		// 获取JSON字段的值
		number, _ := strconv.ParseInt(data["number"].(string), 10, 64)

		color.Green("查询%d号区块请求", number)

		b := bcs.GetBlockChain().GetBlockByNumber(number)
		if b == nil {
			http.Error(w, "此区块号不存在", http.StatusBadRequest)
			return
		}
		m, _ := b.MarshalJSON()

		w.Header().Add("Content-Type", "application/json")
		io.WriteString(w, string(m[:]))

	default:
		log.Printf("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) GetBlockByHash(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:

		var data map[string]interface{}
		// 解析JSON数据

		err := json.NewDecoder(req.Body).Decode(&data)
		if err != nil {
			http.Error(w, "无法解析JSON数据", http.StatusBadRequest)
			return
		}
		// 获取JSON字段的值
		var hash [32]byte
		hashStr, _ := data["hash"].(string)
		ph, _ := hex.DecodeString(hashStr)
		copy(hash[:], ph[:32])
		color.Green("通过哈希查询区块：%x\n", hash)

		b := bcs.GetBlockChain().GetBlockByHash(hash)
		if b == nil {
			http.Error(w, "此区块不存在", http.StatusBadRequest)
			return
		}
		m, _ := b.MarshalJSON()

		w.Header().Add("Content-Type", "application/json")
		io.WriteString(w, string(m[:]))

	default:
		log.Printf("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) GetTxByHash(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:

		var data map[string]interface{}
		// 解析JSON数据

		err := json.NewDecoder(req.Body).Decode(&data)
		if err != nil {
			http.Error(w, "无法解析JSON数据", http.StatusBadRequest)
			return
		}
		// 获取JSON字段的值
		var txHash [32]byte
		txHashStr, _ := data["txHash"].(string)
		ph, _ := hex.DecodeString(txHashStr)
		copy(txHash[:], ph[:32])
		color.Green("通过哈希查询交易：%x\n", txHash)

		tx := bcs.GetBlockChain().GetTransactionByHash(txHash)
		if tx == nil {
			http.Error(w, "此区块不存在", http.StatusBadRequest)
			return
		}
		m, _ := tx.MarshalJSON()

		w.Header().Add("Content-Type", "application/json")
		io.WriteString(w, string(m[:]))

	default:
		log.Printf("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}

//-------------------Server Run And Stop------------------

func (bcs *BlockchainServer) Run() {
	isReady := bcs.SetBlockchainFromFile()
	if isReady {
		bcs.GetBlockChain().Run()
	}

	http.HandleFunc("/", bcs.GetChain)
	http.HandleFunc("/transactions", bcs.Transactions) //GET 方式和 POST方式
	http.HandleFunc("/mine", bcs.Mine)
	http.HandleFunc("/mine/start", bcs.StartMine)
	http.HandleFunc("/mine/stop", bcs.StopMine)
	http.HandleFunc("/amount", bcs.Amount)
	http.HandleFunc("/consensus", bcs.Consensus)
	http.HandleFunc("/history", bcs.History)
	http.HandleFunc("/getBlockByNumber", bcs.GetBlockByNumber)
	http.HandleFunc("/getBlockByHash", bcs.GetBlockByHash)
	http.HandleFunc("/getTxByHash", bcs.GetTxByHash)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(int(bcs.Port())), nil))
}

func (bcs *BlockchainServer) Close() {
	// 在关闭节点之前保存数据到本地
	err := writeCacheToFile(bcs.GetBlockChain(), bcs.filePath)
	color.HiGreen("监测到%d节点已停止！数据已保存至%d.json", bcs.GetBlockChain().Port(), bcs.GetBlockChain().Port())
	if err != nil {
		log.Printf("数据保存失败！: %v", err)
	}
}

// -----------------File Operation---------------------

func writeCacheToFile(cache *block.Blockchain, filePath string) error {
	file, err := os.Create(filePath) // 创建文件，如果文件已存在，会被清空内容
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(cache)
	if err != nil {
		return err
	}

	return nil
}

func loadCacheFromFile(filePath string) (*block.Blockchain, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var cache *block.Blockchain
	err = decoder.Decode(&cache)
	if err != nil {
		return nil, err
	}

	return cache, nil
}
