# go_websocket_ez

簡易的 websocket 搭配 channel 的模型

# url

ws://localhost:8080/user

# 登入封包

{
"cmd": "login",
"data": "test"
}

# 遊戲封包

{
"cmd": "play",
"data": "test"
}

# 透過 channel 關閉某個用戶連線

{
"cmd": "close_server",
"data": "test"
}
