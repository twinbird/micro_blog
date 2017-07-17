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
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/logout", needLogin(logoutHandler))
	http.HandleFunc("/signup", signupHandler)
	http.HandleFunc("/timeline", needLogin(timelineHandler))

	log.Println("Booting up localhost" + port)
	err = http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal(err)
	}
}

// 認証処理
func needLogin(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 認証処理
		if ok, _, _ := sessionManager.IsSessionStarted(w, r); ok == true {
			fn(w, r)
		} else {
			// ダメなら全部indexへ回す
			http.Redirect(w, r, "/", http.StatusFound)
		}
	}
}

// [/timeline]処理用のハンドラ
func timelineHandler(w http.ResponseWriter, r *http.Request) {
	err := responseTemplate.ExecuteTemplate(w, "timeline.tmpl", nil)
	if err != nil {
		log.Println(err)
		http.Error(w, "Sorry.", http.StatusInternalServerError)
	}
}

// [/login]処理用のハンドラ
func loginHandler(w http.ResponseWriter, r *http.Request) {
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
func logoutHandler(w http.ResponseWriter, r *http.Request) {
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
func signupHandler(w http.ResponseWriter, r *http.Request) {
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
