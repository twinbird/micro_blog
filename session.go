package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Session は1ユーザとのセッションを管理するためのオブジェクト
type Session struct {
	sessionID  string
	expireTime time.Time
	data       map[interface{}]interface{}
	lock       sync.Mutex
}

// Set はセッションに対しデータを設定します
func (s *Session) Set(key interface{}, value interface{}) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.data[key] = value
	return nil
}

// Get はセッションからkeyに一致するデータを取得します
func (s *Session) Get(key interface{}) (interface{}, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	v, _ := s.data[key]
	return v, nil
}

// Delete はセッションからkeyに一致するデータを削除します
func (s *Session) Delete(key interface{}) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.data, key)
	return nil
}

// SessionID はセッションのIDを返します
func (s *Session) SessionID() string {
	return s.sessionID
}

// SessionManager はセッション全体を管理するオブジェクト
type SessionManager struct {
	cookieName string
	sessions   map[string]*Session
	lock       sync.Mutex
	maxAge     int
}

// NewSessionManager は新しいセッションマネージャを生成して返す
func NewSessionManager(cookieName string, maxAge int) (*SessionManager, error) {
	mgr := &SessionManager{
		cookieName: cookieName,
		sessions:   make(map[string]*Session),
		maxAge:     maxAge,
	}
	return mgr, nil
}

// IsSessionStarted はセッションを開始していればtrueと現在のセッションオブジェクトを返す.
// セッションを開始していない場合はfalseとnilを返す.
func (mgr *SessionManager) IsSessionStarted(w http.ResponseWriter, r *http.Request) (bool, *Session, error) {
	// ID取得を平行でやるとまずいので
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	// 既存セッションの確認
	cookie, err := r.Cookie(mgr.cookieName)

	if err != nil || cookie.Value == "" {
		// セッションがまだ構築されていない
		return false, nil, nil
	}
	// セッションIDあり
	sid, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		return false, nil, err
	}

	// セッションチェック
	v, ok := mgr.sessions[sid]

	// セッションがなかった
	if ok == false {
		return false, nil, nil
	}

	// あった
	return true, v, nil
}

// 32文字のセッションIDを生成して返却する
func (mgr *SessionManager) sessionID() (string, error) {
	buf := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", buf), nil
}

// SessionStart はセッションを開始してSessionオブジェクトを返す.
func (mgr *SessionManager) SessionStart(w http.ResponseWriter, r *http.Request) (*Session, error) {
	// ID取得を平行でやるとまずいので
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	// セッションIDを取得
	sid, err := mgr.sessionID()
	if err != nil {
		return nil, err
	}

	// クッキーへ設定
	cookie := http.Cookie{
		Name:     mgr.cookieName,
		Value:    url.QueryEscape(sid),
		Path:     "/",
		HttpOnly: true,
	}
	http.SetCookie(w, &cookie)

	// Sessionオブジェクトを作成してマネージャへ保存
	s := &Session{
		sessionID:  sid,
		expireTime: time.Now().Add(time.Duration(mgr.maxAge) * time.Second),
		data:       make(map[interface{}]interface{}),
	}
	mgr.sessions[sid] = s

	return s, nil
}

// SessionEnd は現在のリクエストのセッションを終了する
func (mgr *SessionManager) SessionEnd(w http.ResponseWriter, r *http.Request) error {
	// 現在のリクエストのクッキー取得
	cookie, err := r.Cookie(mgr.cookieName)
	if err != nil || cookie.Value == "" {
		return err
	}
	// ID取得を平行でやるとまずいので
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	// マネージャからセッションオブジェクトを取り除く
	sid, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		return err
	}
	delete(mgr.sessions, sid)

	// セッションを破棄したいので有効期限を現在にする
	newCookie := http.Cookie{
		Name:     mgr.cookieName,
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Now(),
		MaxAge:   -1,
	}
	http.SetCookie(w, &newCookie)

	return nil
}

// GC はセッション情報を定期的に削除するタイマーを起動します
// タイマー間隔はSessionManagerのmaxAgeです
func (mgr *SessionManager) GC() {
	// ID取得を平行でやるとまずいので
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	// 現在時刻
	t := time.Now()

	// 現在のセッションを全て確認し, 生存期間を超えているものを削除
	for k, s := range mgr.sessions {
		if t.After(s.expireTime) == true {
			delete(mgr.sessions, k)
		}
	}

	// 次の実行を設定
	time.AfterFunc(time.Duration(mgr.maxAge), func() { mgr.GC() })
}
