package main

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/olahol/melody"
)

// GopherInfoは、gopherの情報を表す構造体
type GopherInfo struct {
	ID, X, Y string
}

func main() {
	// melodyのインスタンスを作成
	m := melody.New()

	// /wsにアクセスしたときにindex.htmlを返す
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	// /wssにアクセスしたときにwebsocketのハンドシェイクを行う
	http.HandleFunc("/wss", func(w http.ResponseWriter, r *http.Request) {
		m.HandleRequest(w, r)
	})

	// websocketの接続時の処理
	m.HandleConnect(func(s *melody.Session) {
		// すでに接続しているセッションの情報を取得
		ss, _ := m.Sessions()

		// すでに接続しているセッションの情報をすべて取得して、
		for _, o := range ss {
			// infoというキーで保存されている値を取得
			value, exists := o.Get("info")

			// もしinfoというキーで保存されている値がなければ、次のループに移る
			if !exists {
				continue
			}

			// infoというキーで保存されている値をGopherInfo型にキャスト
			info := value.(*GopherInfo)

			// すでに接続しているセッションに対して、新しく接続したセッションの情報を送信
			s.Write([]byte("set " + info.ID + " " + info.X + " " + info.Y))
		}

		// 新しく接続したセッションに対して、idを生成
		id := uuid.NewString()

		// 新しく接続したセッションに対して、idを送信
		s.Set("info", &GopherInfo{id, "0", "0"})

		// すべてのセッションに対して、新しく接続したセッションの情報を送信
		s.Write([]byte("iam " + id))
	})

	// websocketの切断時の処理
	m.HandleDisconnect(func(s *melody.Session) {
		// 切断したセッションの情報を取得
		value, exists := s.Get("info")

		// もし切断したセッションの情報がなければ、次の処理に移る
		if !exists {
			return
		}

		// 切断したセッションの情報をGopherInfo型にキャスト
		info := value.(*GopherInfo)

		// すべてのセッションに対して、切断したセッションの情報を送信
		m.BroadcastOthers([]byte("dis "+info.ID), s)
	})

	// websocketでメッセージを受信したときの処理
	m.HandleMessage(func(s *melody.Session, msg []byte) {
		// 受信したメッセージをスペースで分割
		p := strings.Split(string(msg), " ")

		value, exists := s.Get("info")

		// もし受信したメッセージが2つでなければ、次の処理に移る
		if len(p) >= 2 && exists {
			info := value.(*GopherInfo)
			info.X = p[0]
			info.Y = p[1]

			m.BroadcastOthers([]byte("set "+info.ID+" "+info.X+" "+info.Y), s)
		} else {
			m.Broadcast([]byte("chat " + string(msg)))
		}
	})

	// 5000番ポートでサーバーを起動
	http.ListenAndServe(":5000", nil)
}
