package main

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"regexp"
	"time"
	"unicode/utf8"
)

const (
	// SaltLength はDBへ登録するソルトの長さ
	SaltLength = 30
)

// User はDB登録と画面表示データの引き渡しに使うユーザ情報の構造体
type User struct {
	ID              int64    // 登録したID
	Name            string   // 表示ユーザ名
	Email           string   // 登録Emailアドレス(ID兼ねる)
	Password        string   // パスワード
	ConfirmPassword string   // 確認パスワード
	Salt            string   // ハッシュ化に用いたソルト
	HashedPassword  string   // ハッシュ化されたパスワード
	Messages        []string // エラーメッセージ
}

// n文字のソルトを生成
func createSalt(n int) (string, error) {
	buf := make([]byte, n/2)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", buf), nil
}

// パスワードをハッシュ化
func passwordHashing(password string, salt string) string {
	str := password + salt
	return fmt.Sprintf("%x", sha256.Sum256([]byte(str)))
}

// メールアドレスチェック
// 正しいのか怪しいけど.適当なサイト回ったのにアレンジ.
func validateEmailFormat(email string) bool {
	re := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,10}$`)
	return re.MatchString(email)
}

// UserIDが存在する場合はtrue
func isExistUserID(id int64) (bool, error) {
	// コネクション取得
	db, err := DBConnection()
	if err != nil {
		return false, err
	}

	// クエリ発行
	var dbID int64
	err = db.QueryRow(`
	SELECT
		u.id
	FROM
		users u
	WHERE
		u.id = ?
	`, id).Scan(&dbID)

	// 存在判定
	switch {
	case err == sql.ErrNoRows:
		return false, nil
	case err != nil:
		return false, err
	default:
		return true, nil
	}
}

// emailが存在する場合はtrue
func isExistEmail(email string) (bool, error) {
	// コネクション取得
	db, err := DBConnection()
	if err != nil {
		return false, err
	}

	// クエリ発行
	var dbEmail string
	err = db.QueryRow(`
	SELECT
		u.email
	FROM
		users u
	WHERE
		u.email = ?
	`, email).Scan(&dbEmail)

	// 存在判定
	switch {
	case err == sql.ErrNoRows:
		return false, nil
	case err != nil:
		return false, err
	default:
		return true, nil
	}
}

// emailでユーザを探して, uの内容を置き換える
func (u *User) findByEmail(email string) (bool, error) {
	// コネクション取得
	db, err := DBConnection()
	if err != nil {
		return false, err
	}

	// クエリ発行
	var dbID int64
	var dbName string
	var dbEmail string
	var dbSalt string
	var dbHashedPassword string
	err = db.QueryRow(`
	SELECT
		u.id,
		u.name,
		u.email,
		u.salt,
		u.hashed_password
	FROM
		users u
	WHERE
		u.email = ?
	`, email).Scan(&dbID, &dbName, &dbEmail, &dbSalt, &dbHashedPassword)

	// 存在判定
	switch {
	case err == sql.ErrNoRows:
		return false, nil
	case err != nil:
		return false, err
	default:
		// 見つかったのでデータを設定
		u.ID = dbID
		u.Name = dbName
		u.Salt = dbSalt
		u.HashedPassword = dbHashedPassword
		return true, nil
	}
}

// Validate はDB登録前のバリデーションチェック
func (u *User) Validate() error {
	var messages []string

	// Name
	// 文字数チェック
	if n := utf8.RuneCountInString(u.Name); n < 1 || 30 < n {
		messages = append(messages, "ユーザ名は1文字以上, 30文字以内で入力してください")
	}

	// Email
	if n := utf8.RuneCountInString(u.Email); n < 1 || 50 < n {
		// 文字数チェック
		messages = append(messages, "メールアドレスは1文字以上, 50文字以内のものを利用してください")
	} else if validateEmailFormat(u.Email) == false {
		// 妥当性チェック
		messages = append(messages, "メールアドレスが不正です")
	} else {
		// 登録済みチェック
		exist, err := isExistEmail(u.Email)
		if err != nil {
			return err
		}
		if exist == true {
			messages = append(messages, "登録済みのメールアドレスです")
		}
	}

	// Password
	if n := utf8.RuneCountInString(u.Password); n < 1 || 20 < n {
		// 文字数チェック
		messages = append(messages, "パスワードは8～20文字以内で指定してください")
	} else if u.Password != u.ConfirmPassword {
		// 確認パスワードチェック
		messages = append(messages, "パスワードと確認パスワードが異なります")
	}

	// エラーメッセージを登録しておく
	u.Messages = messages

	return nil
}

// Entry はDBへユーザ情報を新規登録するメソッド
func (u *User) Entry() error {
	// コネクション取得
	db, err := DBConnection()
	if err != nil {
		return err
	}

	// プリペアードステートメント生成
	stmt, err := db.Prepare("INSERT INTO users(name, email, hashed_password, salt, created_at) VALUES(?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// ソルト生成
	salt, err := createSalt(SaltLength)
	if err != nil {
		return err
	}

	// パスワードハッシュ化
	hashedPass := passwordHashing(u.Password, salt)

	// クエリ発行
	result, err := stmt.Exec(u.Name, u.Email, hashedPass, salt, time.Now())
	if err != nil {
		return err
	}
	// 登録したIDを構造体へ入れてやる
	insertID, err := result.LastInsertId()
	if err != nil {
		return err
	}
	u.ID = insertID

	return nil
}

// Authenticate はu内の情報で認証を行う
// 認証成功の場合, uの各フィールドへ値を設定する
func (u *User) Authenticate() (bool, error) {
	findUser := &User{Email: u.Email}
	// User情報をDBから取得
	exist, err := findUser.findByEmail(findUser.Email)
	if err != nil {
		return false, err
	}
	if exist == false {
		u.Messages = append(u.Messages, "メールアドレスまたはパスワードが異なります")
		return false, nil
	}
	// salt + passwordで計算
	hashedPass := passwordHashing(u.Password, findUser.Salt)
	// hashed_passwordと比較
	if hashedPass != findUser.HashedPassword {
		u.Messages = append(u.Messages, "メールアドレスまたはパスワードが異なります")
		return false, nil
	}
	// 呼び出し元へ値をコピー
	u.ID = findUser.ID
	u.Name = findUser.Name
	return true, nil
}

// FollowerForTemplate はフォロアー検索画面表示制御用の構造体
type FollowerForTemplate struct {
	Followers []Follower
}

// Follower は画面表示用のフォロアー情報
type Follower struct {
	ID        int64
	Name      string
	Following bool
}

// queryに部分一致するユーザ一覧を返す
func findFollowersByQuery(uid int64, query string, limit int, offset int) ([]Follower, error) {
	// コネクション取得
	db, err := DBConnection()
	if err != nil {
		return nil, err
	}
	// SQL発行
	rows, err := db.Query(`
		SELECT
			u.id,
			u.name,
			CASE WHEN f.follower_id IS NULL THEN 'NO_FOLLOWING'
			ELSE 'FOLLOWING'
			END AS follow_status
		FROM
			users u
		LEFT JOIN
			followers f
		ON
			f.follower_id = u.id
		AND
			f.user_id = ?
		WHERE
			u.name LIKE '%' || ? || '%'
		ORDER BY
			u.created_at desc
		LIMIT ?
		OFFSET ?
	`, uid, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	followers := make([]Follower, 0)
	for rows.Next() {
		var f Follower
		var followStatus string
		if err := rows.Scan(&f.ID, &f.Name, &followStatus); err != nil {
			return nil, err
		}
		if followStatus == "NO_FOLLOWING" {
			f.Following = false
		} else {
			f.Following = true
		}
		followers = append(followers, f)
	}
	return followers, nil
}
