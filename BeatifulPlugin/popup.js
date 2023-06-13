$(document).ready(function () {

    {


        const homeButton = $("#nav-home-tab"); //切换信息主页
        const transferButton = $("#nav-profile-tab"); //切换转账页
        const txConfirmButton = $("#buttonSubmit");//转账确认按钮
        const historyButton = $("#nav-contact-tab"); //切换交易历史页
        const addButton = $("#nav-add-tab");//切换添加账号页
        const readButton = $("#readButton"); //加载私钥按钮
        const writeButton = $("#writeButton"); //随机生成按钮
        const removeButton = $("#removeButton");//清空账号函数
        const saveButton = $('#saveAccount');//保存账号按钮
        var $selectElement = $("#accountSelect");//账号下拉框

        //时间戳格式化
        function formatDate(timestamp) {
            var milliseconds = Math.floor(timestamp / 1000000); // 将纳秒转换为毫秒
            var date = new Date(milliseconds);
            return date.toLocaleString('zh-CN', {
                year: 'numeric',
                month: '2-digit',
                day: '2-digit',
                hour: '2-digit',
                minute: '2-digit',
                second: '2-digit'
            });
        }


        loadSelectElement()//加载账号列表
        function clearAllAccounts() {
            chrome.storage.local.clear(function () {
                if (chrome.runtime.lastError) {
                    console.error(chrome.runtime.lastError);
                } else {
                    customAlert("存储在插件的账号已全部清除");
                }
                initHomeInputs()
                loadSelectElement()
            });
        }


        function loadAccountInfo(privateKey) {
            let blockchainAddress, publicKey

            //加载账户信息
            $.ajax({
                url: "http://127.0.0.1:8080/walletByPrivateKey", type: "POST", data: {
                    privateKey: privateKey, // 添加其他参数...
                }, success: function (response) {
                    $("#secondPublicKey").val(response["public_key"]);
                    $("#firstPrivateKey").val(response["private_key"]);
                    $("#thirdBlockChainAddress").val(response["blockchain_address"]);
                    blockchainAddress = response["blockchain_address"]
                    publicKey = response["public_key"]
                    let postData = {
                        blockchain_address: blockchainAddress,
                    };
                    $.ajax({
                        url: "http://127.0.0.1:8080/wallet/amount",
                        type: "POST",
                        //contentType: "application/json",
                        data: JSON.stringify(postData),
                        success: function (response) {
                            $('#wallet_amount').text('$' + response["amount"]);
                            console.info(response);
                        },
                        error: function (error) {
                            console.error(error);
                        },
                    });
                }, error: function (error) {
                    console.error(error);
                },
            });
        }

        function loadHistoryTxs() {

            const dataExample = [
                {
                    "tx_hash": "e382606d4de7c87d21f7ae6d2d78450298196c11e85243f738fe83f739494573",
                    "sender_blockchain_address": "3RHdMc9vqDoiPZ9bQGQEgQkpQUry33A1mAMPz4Y3f89L",
                    "recipient_blockchain_address": "BqnoiMm4Pot5EDGjV3yR7rKXvuTaQbn1qn8zkykJnjQ9",
                    "value": 200,
                    "number": 1,
                    "timestamp": '2006-01-02 15:04:05'
                },
            ];

            var blockchainAddress = $("#thirdBlockChainAddress").val();
            let postData = {
                blockchain_address: blockchainAddress,
            };
            $.ajax({
                url: "http://127.0.0.1:8080/wallet/history",
                method: "POST",
                data: JSON.stringify(postData),
                success: function (response) {
                    var transactionData = response.completedTransactions;
                    var accordion = $('#transactionAccordion');

                    // 清空交易卡片容器
                    accordion.empty();
                    // 使用循环倒序插入交易卡片
                    $.each(transactionData.reverse(), function (index, transaction) {
                        var truncatedHash = transaction.tx_hash.substring(0, 20) + '...' + transaction.tx_hash.substring(transaction.tx_hash.length - 11);
                        var card = $('<div class="card">');
                        var cardHeader = $('<div class="card-header">');
                        var cardTitle = $('<h2 class="mb-0">');
                        var cardButton = $('<button class="btn btn-link" type="button" data-bs-toggle="collapse">')
                            .attr('data-bs-target', '#transactionCollapse' + index)
                            .attr('aria-expanded', 'false')
                            .text(truncatedHash)
                            .val(transaction.tx_hash)
                        var cardCollapse = $('<div class="collapse">')
                            .attr('id', 'transactionCollapse' + index)
                            .attr('data-bs-parent', '#transactionAccordion');
                        var cardBody = $('<div class="card-body">');
                        var cardContent = $('<ul>').addClass('list-group')
                            .append($('<li>').addClass('list-group-item').text('交易哈希: ' + transaction.tx_hash))
                            .append($('<li>').addClass('list-group-item').text('发送方地址: ' + transaction.sender_blockchain_address))
                            .append($('<li>').addClass('list-group-item').text('接收方地址: ' + transaction.recipient_blockchain_address))
                            .append($('<li>').addClass('list-group-item').text('金额: ' + transaction.value))
                            .append($('<li>').addClass('list-group-item').text('所在区块号: ' + transaction.number))
                            .append($('<li>').addClass('list-group-item').text('确认时间: ' + formatDate(transaction.timestamp)));


                        cardContent.find('li').each(function () {
                            var liText = $(this).text();
                            if (liText.includes(blockchainAddress) && blockchainAddress !== '') {
                                $(this).html($(this).text() + '<span class="badge bg-primary" style="margin-left: 4px;"> 我</span>');
                            }
                        });

                        cardTitle.append(cardButton);
                        cardHeader.append(cardTitle);
                        cardCollapse.append(cardBody.append(cardContent));
                        card.append(cardHeader, cardCollapse);
                        card.addClass('mb-3'); // 添加边距类
                        accordion.append(card);

                    });
                },
                error: function (xhr, status, error) {
                    // 处理请求错误
                    console.log("请求错误:", error);
                }
            });


        }

        function loadSelectElement() {
            chrome.storage.local.get('privateKeys', function (data) {
                if (chrome.runtime.lastError) {
                    console.error(chrome.runtime.lastError);
                    return;
                }

                var storedPrivateKeys = data.privateKeys;
                $selectElement.empty();
                $selectElement.append($("<option disabled selected>").text("选择账号"));
                // 遍历私钥对象
                for (var account in storedPrivateKeys) {
                    if (storedPrivateKeys.hasOwnProperty(account)) {
                        var privateKey = storedPrivateKeys[account];

                        // 创建新的选项元素
                        var optionElement = $("<option>")
                            .val(privateKey)
                            .text(account);

                        // 将选项元素添加到下拉选项中
                        $selectElement.append(optionElement);
                    }
                }
                console.log("已加载下拉框")
            });
        }

        function addAccount(privateKey) {
            chrome.storage.local.get('privateKeys', function (data) {
                if (chrome.runtime.lastError) {
                    console.error(chrome.runtime.lastError);
                    return;
                }

                var existingPrivateKeys = data.privateKeys || {}; // 获取现有的私钥列表
                var newPrivateKey = privateKey; // 替换为实际的新私钥值


                // 将新的私钥追加到现有的私钥对象中
                var newKey = "Account" + (Object.keys(existingPrivateKeys).length); // 生成新的唯一键

                //币池（交易不验证随意发）
                if (privateKey === '6666') {
                    newKey = "币池（交易不验证）"
                }
                existingPrivateKeys[newKey] = newPrivateKey; // 将新的私钥追加到现有的私钥对象中
                // 将更新后的私钥对象存回 chrome.storage
                chrome.storage.local.set({'privateKeys': existingPrivateKeys}, function () {
                    if (chrome.runtime.lastError) {
                        console.error(chrome.runtime.lastError);
                        return;
                    }
                    console.log("新私钥已成功追加到chrome.storage.");
                    //弹出提示框
                    customAlert(newKey + "添加成功!")
                    loadSelectElement()
                });
            });

        }

        //初始化添加表单
        function initAddInputs() {
            $("#inputPrivateKey").val("");
            $("#inputPublic").val("");
            $("#blockchain_address").val("");
            saveButton.prop("disabled", true);
        }

        //初始化账户信息表单
        function initHomeInputs() {
            $("#wallet_amount").text("$0")
            $("#firstPrivateKey").val("");
            $("#secondPublicKey").val("");
            $("#thirdBlockChainAddress").val("");
        }

        //弹出自定义内容的提示框
        function customAlert(textContend) {
            $('#customAlert').text(textContend); // 设置弹出框的文本内容
            $('#customAlert').fadeIn(300, function () {
                setTimeout(function () {
                    $('#customAlert').fadeOut(300);
                }, 2000); // 设置提示框显示时间，这里是2秒
            });
        }

        //监听账号切换，重载信息
        $selectElement.on('change', function () {


            //获取所有文本内容
            var text = $selectElement.text();

            //获取选中的账号名称
            var selectedAccount = $selectElement.find("option:selected").text();

            //获取选中的privateKey值
            var privateKey = $selectElement.val();

            //获取选中的option的索引值
            var index = $selectElement.get(0).selectedIndex;
            //customAlert("已切换为" + selectedAccount);
            console.log(text);
            console.log(selectedAccount);
            console.log(privateKey);
            console.log(index);
            loadAccountInfo(privateKey)
            homeButton.click() //切换账号切回主页

        });

        //确认转账按钮
        txConfirmButton.on("click", function () {
            var privateKey = $selectElement.val();
            //载入账户
            loadAccountInfo(privateKey)
            let transaction_data = {
                sender_private_key: $("#firstPrivateKey").val(),
                sender_blockchain_address: $("#thirdBlockChainAddress").val(),
                recipient_blockchain_address: $("#inputReceiveAddress").val(),
                sender_public_key: $("#secondPublicKey").val(),
                value: $("#inputAmount").val(),
            };
            let confirm_text = "确定要发送吗?\n" + "发送者:\n" + transaction_data.sender_blockchain_address + "\n接收者:" + transaction_data.recipient_blockchain_address + "\n金额:" + transaction_data.value;
            let confirm_result = confirm(confirm_text);
            if (confirm_result) {
                $.ajax({
                    url: "http://127.0.0.1:8080/transaction",
                    type: "POST",
                    contentType: "application/json",
                    data: JSON.stringify(transaction_data),
                    success: function (response) {
                        console.info("response:", response);
                        console.info("response.message:", response.message);

                        if (response.message === "fail") {
                            customAlert("failed 222");
                            return;
                        }

                        customAlert("发送成功" + JSON.stringify(response));
                    },
                    error: function (response) {
                        console.error(response);
                        customAlert("发送失败");
                    },
                });
            }


        });
        //用已有私钥加载
        readButton.on("click", function () {
            var inputValue = prompt("请输入PrivateKey(私钥)值：");
            if (inputValue !== null) {
                $("#inputPrivateKey").val(inputValue);
            }
            //用已有私钥加载账号
            $.ajax({
                url: "http://127.0.0.1:8080/walletByPrivateKey",
                type: "POST",
                data: {
                    privateKey: inputValue, // 添加其他参数...
                },
                success: function (response) {
                    $("#inputPublic").val(response["public_key"]);
                    $("#inputPrivateKey").val(response["private_key"]);
                    $("#blockchain_address").val(response["blockchain_address"]);
                    console.info(response);
                }, error: function (error) {
                    console.error(error);
                },
            });
            saveButton.prop ("disabled", false);
        });


        //随机生成新账号按钮
        writeButton.on("click", function () {
            //随机生成账号
            $.ajax({
                url: "http://127.0.0.1:8080/wallet", type: "POST", success: function (response) {
                    $("#inputPublic").val(response["public_key"]);
                    $("#inputPrivateKey").val(response["private_key"]);
                    $("#blockchain_address").val(response["blockchain_address"]);
                    console.info(response);
                }, error: function (error) {
                    console.error(error);
                },
            });
            saveButton.prop("disabled", false);
        });

        //切换账号信息nav时刷新最新余额
        homeButton.click(function () {
            var privateKey = $selectElement.val();
            loadAccountInfo(privateKey)
        })
        //切换交易记录nav渲染交易记录
        historyButton.click(function () {
            loadHistoryTxs();
        })
        //切换添加用户nav时初始化表单
        addButton.click(function () {
            initAddInputs()
        });

        //点击确认保存按钮时，添加账号，初始化表单
        saveButton.on("click", function () {
            var privateKeyValue = $("#inputPrivateKey").val();
            addAccount(privateKeyValue);
            initAddInputs();
        })
        removeButton.on("click", function () {
            if (confirm("确定删除？")) {
                clearAllAccounts()
            }
        })

    }
});
