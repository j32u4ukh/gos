# gos
Golang server

## v1.1.0
* 修正斷線重連 與 斷線，兩者過程中的資源釋放與變數重置
* 允許透過傳遞 site(server id) 和 cid(連線 ID)，將 A 伺服器的任務轉交給 B 伺服器處理，結束後再返還給 A 伺服器，並回覆客戶端