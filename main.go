package main

import (
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
)

// 各ページのテンプレート入りテンプレート
var responseTemplate *template.Template

// セッションマネージャ
var sessionManager *SessionManager

const (
	// SessionUserIDKey はSessionManagerのSession内におけるUserIDのキー
	SessionUserIDKey = "UserID"
	// TimelinePageLimit は1ページあたりのsweetの表示件数
	TimelinePageLimit = 50
)

// テンプレートファイルを読み込む
func loadTemplates(dir string) (*template.Template, error) {
	// templateディレクトリ以下のテンプレートファイルを全てパースしておく
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var filepaths []string
	for _, info := range files {
		if info.IsDir() == false {
			path := filepath.Join(dir, info.Name())
			filepaths = append(filepaths, path)
		}
	}
	return template.ParseFiles(filepaths...)
}

func init() {
	var err error

	// テンプレート初期化
	responseTemplate, err = loadTemplates("./templates")
	if err != nil {
		log.Fatal(err)
	}

	// セッションマネージャ初期化
	sessionManager, err = NewSessionManager("suitter", 86400)
	if err != nil {
		log.Fatal(err)
	}
	sessionManager.GC()

	// アプリケーション設定の読み込み
	applicationConfig, err = loadConfig("./")
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	var err error
	var port string
	port = ":80"

	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/login", unneedLogin(loginHandler))
	http.HandleFunc("/logout", needLogin(logoutHandler))
	http.HandleFunc("/signup", unneedLogin(signupHandler))
	http.HandleFunc("/timeline", needLogin(timelineHandler))
	http.HandleFunc("/sweets", needLogin(sweetsHandler))

	log.Println("Booting up localhost" + port)
	err = http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal(err)
	}
}

// HandlerFuncWithSession は認証をかませるためにHandlerFuncを拡張したもの
type HandlerFuncWithSession func(http.ResponseWriter, *http.Request, *Session)

// 認証処理
func needLogin(fn HandlerFuncWithSession) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 認証処理
		if ok, s, err := sessionManager.IsSessionStarted(w, r); ok == true {
			fn(w, r, s)
		} else {
			// errorがあればロギング
			if err != nil {
				log.Println(err)
			}
			// ダメなら全部indexへ回す
			http.Redirect(w, r, "/", http.StatusFound)
		}
	}
}

// 認証が不要な場合のラッパー
func unneedLogin(fn HandlerFuncWithSession) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ok, s, err := sessionManager.IsSessionStarted(w, r); ok == true && err == nil {
			// 認証ができればセッションも渡してやる
			fn(w, r, s)
		} else if err != nil {
			log.Println(err)
			fn(w, r, nil)
		} else {
			// 認証が出来なければnil渡す
			fn(w, r, nil)
		}
	}
}

// [/sweets]処理用のハンドラ
func sweetsHandler(w http.ResponseWriter, r *http.Request, s *Session) {
	switch r.Method {
	case "POST":
		// 認証したユーザのIDを取得
		uidv, err := s.Get(SessionUserIDKey)
		if err != nil {
			log.Println(err)
			http.Error(w, "Sorry.", http.StatusInternalServerError)
			return
		}
		uid, ok := uidv.(int64)
		if ok == false {
			log.Println("user_id type assertion fail")
			http.Error(w, "Sorry.", http.StatusInternalServerError)
			return
		}
		// Postパラメータを取得して投稿データを作成
		if err := r.ParseForm(); err != nil {
			log.Println(err)
			http.Error(w, "Sorry.", http.StatusInternalServerError)
			return
		}
		post := &Post{
			UserID:  uid,
			Message: r.PostFormValue("message"),
		}
		// 入力チェック
		if err := post.Validate(); err != nil {
			// sweetsの取得
			posts, err := Sweets(uid, TimelinePageLimit, 0)
			if err != nil {
				log.Println("user_id type assertion fail")
				http.Error(w, "Sorry.", http.StatusInternalServerError)
				return
			}
			timeline := &TimelineForTemplate{Sweets: posts}
			// 入力エラーがあればtimelineの入力フォームを再表示
			err = responseTemplate.ExecuteTemplate(w, "timeline.tmpl", timeline)
			if err != nil {
				log.Println(err)
				http.Error(w, "Sorry.", http.StatusInternalServerError)
			}
			return
		}
		// 登録
		if err := post.Entry(); err != nil {
			if err != nil {
				log.Println(err)
				http.Error(w, "Sorry.", http.StatusInternalServerError)
				return
			}
		}
		// sweetsの取得
		posts, err := Sweets(uid, TimelinePageLimit, 0)
		if err != nil {
			log.Println("user_id type assertion fail")
			http.Error(w, "Sorry.", http.StatusInternalServerError)
			return
		}
		timeline := &TimelineForTemplate{Sweets: posts}
		// timelineの表示
		err = responseTemplate.ExecuteTemplate(w, "timeline.tmpl", timeline)
		if err != nil {
			log.Println(err)
			http.Error(w, "Sorry.", http.StatusInternalServerError)
		}
	default:
		http.NotFound(w, r)
	}
}

// [/timeline]処理用のハンドラ
func timelineHandler(w http.ResponseWriter, r *http.Request, s *Session) {
	// 認証したユーザのIDを取得
	uidv, err := s.Get(SessionUserIDKey)
	if err != nil {
		log.Println(err)
		http.Error(w, "Sorry.", http.StatusInternalServerError)
		return
	}
	uid, ok := uidv.(int64)
	if ok == false {
		log.Println("user_id type assertion fail")
		http.Error(w, "Sorry.", http.StatusInternalServerError)
		return
	}

	// sweetsの取得
	posts, err := Sweets(uid, TimelinePageLimit, 0)
	if err != nil {
		log.Println("user_id type assertion fail")
		http.Error(w, "Sorry.", http.StatusInternalServerError)
		return
	}

	// 表示用データの作成
	timeline := &TimelineForTemplate{Sweets: posts}

	err = responseTemplate.ExecuteTemplate(w, "timeline.tmpl", timeline)
	if err != nil {
		log.Println(err)
		http.Error(w, "Sorry.", http.StatusInternalServerError)
	}
}

// [/login]処理用のハンドラ
func loginHandler(w http.ResponseWriter, r *http.Request, s *Session) {
	switch r.Method {
	case "GET":
		err := responseTemplate.ExecuteTemplate(w, "login.tmpl", nil)
		if err != nil {
			log.Println(err)
			http.Error(w, "Sorry.", http.StatusInternalServerError)
			return
		}
	case "POST":
		// 認証処理
		if err := r.ParseForm(); err != nil {
			log.Println(err)
			http.Error(w, "Sorry.", http.StatusInternalServerError)
			return
		}
		u := &User{
			Email:    r.PostFormValue("email"),
			Password: r.PostFormValue("password"),
		}
		ok, err := u.Authenticate()
		if err != nil {
			log.Println(err)
			http.Error(w, "Sorry.", http.StatusInternalServerError)
			return
		}
		// 認証の確認
		if ok == true {
			// セッションを新規に取得する
			s, err := sessionManager.SessionStart(w, r)
			if err != nil {
				log.Println(err)
				http.Error(w, "Sorry.", http.StatusInternalServerError)
				return
			}
			// セッションへユーザIDを登録
			if err := s.Set(SessionUserIDKey, u.ID); err != nil {
				log.Println(err)
				http.Error(w, "Sorry.", http.StatusInternalServerError)
				return
			}
			// タイムラインへリダイレクトする
			http.Redirect(w, r, "/timeline", http.StatusFound)
		} else {
			// 認証失敗したらメッセージを出して同じページ出してやる
			err := responseTemplate.ExecuteTemplate(w, "login.tmpl", u)
			if err != nil {
				log.Println(err)
				http.Error(w, "Sorry.", http.StatusInternalServerError)
				return
			}
		}
	default:
		http.NotFound(w, r)
	}
}

// [/logout]処理用のハンドラ
func logoutHandler(w http.ResponseWriter, r *http.Request, s *Session) {
	switch r.Method {
	case "POST":
		if err := sessionManager.SessionEnd(w, r); err != nil {
			log.Println(err)
			http.Error(w, "Sorry.", http.StatusInternalServerError)
			return
		}
		// indexへ回す
		http.Redirect(w, r, "/", http.StatusFound)
	default:
		http.NotFound(w, r)
	}
}

// [/signup]処理用のハンドラ
func signupHandler(w http.ResponseWriter, r *http.Request, s *Session) {
	switch r.Method {
	case "GET":
		// ログイン済みならタイムラインへリダイレクトする
		if ok, _, _ := sessionManager.IsSessionStarted(w, r); ok == true {
			http.Redirect(w, r, "/timeline", http.StatusFound)
			return
		}
		err := responseTemplate.ExecuteTemplate(w, "signup.tmpl", nil)
		if err != nil {
			log.Println(err)
			http.Error(w, "Sorry.", http.StatusInternalServerError)
		}
	case "POST":
		// ログイン済みならタイムラインへリダイレクトする
		if ok, _, _ := sessionManager.IsSessionStarted(w, r); ok == true {
			http.Redirect(w, r, "/timeline", http.StatusFound)
			return
		}
		// ユーザを登録する
		if err := r.ParseForm(); err != nil {
			log.Println(err)
			http.Error(w, "Sorry.", http.StatusInternalServerError)
			return
		}
		u := &User{
			Name:            r.PostFormValue("name"),
			Email:           r.PostFormValue("email"),
			Password:        r.PostFormValue("password"),
			ConfirmPassword: r.PostFormValue("confirm_password"),
		}
		// 検証
		err := u.Validate()
		if err != nil {
			log.Println(err)
			http.Error(w, "Sorry.", http.StatusInternalServerError)
			return
		}
		// 入力エラーがあった
		if 0 < len(u.Messages) {
			err := responseTemplate.ExecuteTemplate(w, "signup.tmpl", u)
			if err != nil {
				log.Println(err)
				http.Error(w, "Sorry.", http.StatusInternalServerError)
				return
			}
			return
		}
		// 登録
		if err := u.Entry(); err != nil {
			log.Println(err)
			http.Error(w, "Sorry.", http.StatusInternalServerError)
			return
		}
		// セッションを新規に取得する
		s, err := sessionManager.SessionStart(w, r)
		if err != nil {
			log.Println(err)
			http.Error(w, "Sorry.", http.StatusInternalServerError)
			return
		}
		// セッションへユーザIDを登録
		if err := s.Set(SessionUserIDKey, u.ID); err != nil {
			log.Println(err)
			http.Error(w, "Sorry.", http.StatusInternalServerError)
			return
		}
		// タイムラインへリダイレクトする
		http.Redirect(w, r, "/timeline", http.StatusFound)
	default:
		http.NotFound(w, r)
	}
}
