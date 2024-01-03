package main

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/olahol/melody"
)

// GopherInfoは、gopherの情報を表す構造体
type GopherInfo struct {
	ID, Name, X, Y string
}

// 各ルームごとのセッションを管理するためのマップ
var rooms = make(map[string]map[string]map[*melody.Session]bool)

// ルーム内の他のセッションにメッセージをブロードキャストするヘルパー関数
func sendToRoom(roomID string, msg []byte) {
	if roomSessions, ok := rooms[roomID]; ok {
		if users, exists := roomSessions["users"]; exists {
			for session := range users {
				session.Write(msg)
			}
		}
	}
}

// ルーム内の他のセッションにメッセージをブロードキャストするヘルパー関数（送信者自身を除く）
func sendToRoomOthers(roomID string, msg []byte, sender *melody.Session) {
	if roomSessions, ok := rooms[roomID]; ok {
		if users, exists := roomSessions["users"]; exists {
			for session := range users {
				if session != sender {
					session.Write(msg)
				}
			}
		}
	}
}

func main() {
	// melodyのインスタンスを作成
	m := melody.New()

	// /にアクセスしたときにindex.htmlを返す
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	// /wssにアクセスしたときにwebsocketのハンドシェイクを行う
	http.HandleFunc("/wss", func(w http.ResponseWriter, r *http.Request) {
		m.HandleRequest(w, r)
	})

	// websocketの接続時の処理
	m.HandleConnect(func(s *melody.Session) {
		// URLから取得したルームID（この例ではクエリパラメータとして渡す）
		roomID := s.Request.URL.Query().Get("id")

		// ルームが存在しない場合は新規作成
		if _, ok := rooms[roomID]; !ok {
			rooms[roomID] = make(map[string]map[*melody.Session]bool)
		}

		// ルームごとに独立したマップを作成
		if _, ok := rooms[roomID]["users"]; !ok {
			rooms[roomID]["users"] = make(map[*melody.Session]bool)
		}

		// ルームに参加したセッションを記録
		rooms[roomID]["users"][s] = true

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
			s.Write([]byte("set " + info.ID + " " + info.Name + " " + info.X + " " + info.Y))
		}

		// 新しく接続したセッションに対して、idを生成
		id := uuid.NewString()

		name := ""

		// 新しく接続したセッションを登録
		s.Set("info", &GopherInfo{id, name, "0", "0"})

		// すべてのセッションに対して、新しく接続したセッションの情報を送信
		s.Write([]byte("iam " + id))
	})

	// websocketの切断時の処理
	m.HandleDisconnect(func(s *melody.Session) {
		// URLから取得したルームID（この例ではクエリパラメータとして渡す）
		roomID := s.Request.URL.Query().Get("id")

		// ルームからセッションを削除
		delete(rooms[roomID]["users"], s)

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
		roomID := s.Request.URL.Query().Get("id")

		// メッセージを処理する
		p := strings.Split(string(msg), " ")

		value, exists := s.Get("info")

		// nilチェックを行い、valueがnilの場合の処理を追加する
		if !exists || value == nil {
			// エラーハンドリングや適切な処理を行う
			// 例えばログを出力する、デフォルト値を設定するなど
			return
		}

		// valueがnilでない場合の処理を行う
		info, ok := value.(*GopherInfo)
		if !ok {
			// インターフェース型をGopherInfoに変換できない場合のエラーハンドリングを行う
			return
		}

		// メッセージが"set"で始まる場合の処理
		if p[0] == "set" {
			info.Name = p[1]
			info.X = p[2]
			info.Y = p[3]
			sendToRoomOthers(roomID, []byte("set "+info.ID+" "+info.Name+" "+info.X+" "+info.Y), s)
		} else if p[0] == "chat" {
			name := p[1]
			text := p[2]
			sendToRoom(roomID, []byte("chat "+name+" "+text))
		}
	})

	// 5000番ポートでサーバーを起動
	http.ListenAndServe(":7030", nil)
}
