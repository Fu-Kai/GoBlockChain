package main

import (
	"KaiFuBlockChain/block"
	"KaiFuBlockChain/utils"
	"KaiFuBlockChain/wallet"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

type WalletServer struct {
	port    uint16
	gateway string //区块链的节点地址
}

func NewWalletServer(port uint16, gateway string) *WalletServer {
	return &WalletServer{port, gateway}
}

func (ws *WalletServer) Port() uint16 {
	return ws.port
}

func (ws *WalletServer) Gateway() string {
	return ws.gateway
}

func (ws *WalletServer) Wallet(w http.ResponseWriter, req *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")
	//设置允许的方法
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")

	switch req.Method {
	case http.MethodPost:
		w.Header().Add("Content-Type", "application/json")
		myWallet := wallet.NewWallet()
		m, _ := myWallet.MarshalJSON()
		io.WriteString(w, string(m[:]))
	default:
		w.WriteHeader(http.StatusBadRequest)
		log.Println("ERROR: 非法的HTTP请求方式")
	}
}

func (ws *WalletServer) walletByPrivateKey(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	//设置允许的方法
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")

	switch req.Method {
	case http.MethodPost:

		w.Header().Add("Content-Type", "application/json")
		privateKey := req.FormValue("privateKey")
		myWallet := wallet.LoadWallet(privateKey)
		m, _ := myWallet.MarshalJSON()
		io.WriteString(w, string(m[:]))
	default:
		w.WriteHeader(http.StatusBadRequest)
		log.Println("ERROR: Invalid HTTP Method")
	}
}

type TransactionRequest struct {
	SenderPrivateKey           *string `json:"sender_private_key"`
	SenderBlockchainAddress    *string `json:"sender_blockchain_address"`
	RecipientBlockchainAddress *string `json:"recipient_blockchain_address"`
	SenderPublicKey            *string `json:"sender_public_key"`
	Value                      *string `json:"value"`
}

func (tr *TransactionRequest) Validate() bool {
	if tr.SenderPrivateKey == nil ||
		tr.SenderBlockchainAddress == nil ||
		tr.RecipientBlockchainAddress == nil || strings.TrimSpace(*tr.RecipientBlockchainAddress) == "" ||
		tr.SenderPublicKey == nil ||
		tr.Value == nil || len(*tr.Value) == 0 {
		return false
	}
	return true
}

func (ws *WalletServer) CreateTransaction(
	w http.ResponseWriter,
	req *http.Request) {
	defer req.Body.Close()
	switch req.Method {
	case http.MethodPost:
		var t TransactionRequest
		log.Println("req.Body==", req.Body)
		decoder := json.NewDecoder(req.Body)
		decoder.Decode(&t)
		log.Printf("\n\n\n")
		log.Println("发送人公钥SenderPublicKey ==", *t.SenderPublicKey)
		log.Println("发送人私钥SenderPrivateKey ==", *t.SenderPrivateKey)
		log.Println("发送人地址SenderBlockchainAddress ==", *t.SenderBlockchainAddress)
		log.Println("接收人地址RecipientBlockchainAddress ==", *t.RecipientBlockchainAddress)
		log.Println("金额Value ==", *t.Value)
		log.Printf("\n\n\n")

		publicKey := utils.PublicKeyFromString(*t.SenderPublicKey)
		privateKey := utils.PrivateKeyFromString(*t.SenderPrivateKey, publicKey)
		value, err := strconv.ParseUint(*t.Value, 10, 64)
		if err != nil {
			log.Println("ERROR: parse error")
			io.WriteString(w, string(utils.JsonStatus("fail")))
			return
		}

		if !t.Validate() {
			log.Println("ERROR: missing field(s)")
			io.WriteString(w, string(utils.JsonStatus("Validate fail")))
			return
		}

		w.Header().Add("Content-Type", "application/json")

		// 交易签名
		transaction := wallet.NewTransaction(privateKey, publicKey,
			*t.SenderBlockchainAddress, *t.RecipientBlockchainAddress, value)
		signature := transaction.GenerateSignature()
		signatureStr := signature.String()
		color.Red("signature:%s", signature)

		bt := &block.TransactionRequest{
			SenderBlockchainAddress:    t.SenderBlockchainAddress,
			RecipientBlockchainAddress: t.RecipientBlockchainAddress,
			SenderPublicKey:            t.SenderPublicKey,
			Value:                      &value,
			Signature:                  &signatureStr,
		}
		m, _ := json.Marshal(bt)
		color.Green("提交给BlockServer交易:%s", m)
		buf := bytes.NewBuffer(m)

		resp, _ := http.Post(ws.Gateway()+"/transactions", "application/json", buf)

		if resp.StatusCode == 201 {
			// 201是哪里来的？请参见blockserver  Transactions方法的  w.WriteHeader(http.StatusCreated)语句
			io.WriteString(w, string(utils.JsonStatus("success")))
			return
		}
		io.WriteString(w, string(utils.JsonStatus("fail")))

	default:
		w.WriteHeader(http.StatusBadRequest)
		log.Println("ERROR: 非法的HTTP请求方式")
	}
}

func (ws *WalletServer) WalletAmount(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")

	fmt.Printf("Call WalletAmount  METHOD:%s\n", req.Method)
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
		color.Blue("请求查询账户%s的余额", blockchainAddress)

		// 构建请求数据
		requestData := struct {
			BlockchainAddress string `json:"blockchain_address"`
		}{
			BlockchainAddress: blockchainAddress,
		}

		// 将请求数据编码为JSON
		jsonData, err := json.Marshal(requestData)
		if err != nil {
			fmt.Printf("编码JSON时发生错误:%v", err)
			return
		}

		bcsResp, _ := http.Post(ws.Gateway()+"/amount", "application/json", bytes.NewBuffer(jsonData))

		//返回给客户端
		w.Header().Add("Content-Type", "application/json")
		if bcsResp.StatusCode == 200 {
			decoder := json.NewDecoder(bcsResp.Body)
			var bar block.AmountResponse
			err := decoder.Decode(&bar)
			if err != nil {
				log.Printf("ERROR: %v", err)
				io.WriteString(w, string(utils.JsonStatus("fail")))
				return
			}

			resp_message := struct {
				Message string `json:"message"`
				Amount  uint64 `json:"amount"`
			}{
				Message: "success",
				Amount:  bar.Amount,
			}
			m, _ := json.Marshal(resp_message)
			io.WriteString(w, string(m[:]))
		} else {
			io.WriteString(w, string(utils.JsonStatus("fail")))
		}
	default:
		log.Printf("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (ws *WalletServer) History(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")

	fmt.Printf("Call History  METHOD:%s\n", req.Method)
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
		color.Blue("请求查询账户%s的交易历史", blockchainAddress)

		// 构建请求数据
		requestData := struct {
			BlockchainAddress string `json:"blockchain_address"`
		}{
			BlockchainAddress: blockchainAddress,
		}

		// 将请求数据编码为JSON
		jsonData, err := json.Marshal(requestData)
		if err != nil {
			fmt.Printf("编码JSON时发生错误:%v", err)
			return
		}

		bcsResp, _ := http.Post(ws.Gateway()+"/history", "application/json", bytes.NewBuffer(jsonData))

		//返回给客户端
		w.Header().Add("Content-Type", "application/json")
		if bcsResp.StatusCode == 200 {
			// 将BlockchainServer的响应转发给客户端
			copyHeader(w.Header(), bcsResp.Header)
			w.WriteHeader(bcsResp.StatusCode)
			io.Copy(w, bcsResp.Body)
		} else {
			io.WriteString(w, string(utils.JsonStatus("fail")))
		}
	default:
		log.Printf("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}
func copyHeader(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}
func (ws *WalletServer) Run() {

	http.HandleFunc("/wallet", ws.Wallet)
	http.HandleFunc("/walletByPrivateKey", ws.walletByPrivateKey)
	http.HandleFunc("/transaction", ws.CreateTransaction)
	http.HandleFunc("/wallet/amount", ws.WalletAmount)
	http.HandleFunc("/wallet/history", ws.History)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+strconv.Itoa(int(ws.Port())), nil))
}
