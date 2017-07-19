package main

import (
	"time"
	"unicode/utf8"
)

// Post はメッセージ投稿のメッセージ1つを表す構造体
type Post struct {
	ID        int64
	UserID    int64
	UserName  string
	Message   string
	CreatedAt time.Time
}

// TimelineForTemplate はタイムライン画面用のデータ構造
type TimelineForTemplate struct {
	Messages []string
	Sweets   []Post
}

// Validate はDB登録前のバリデーションチェック
func (p *Post) Validate() error {
	var messages []string

	// UserIDの登録済みチェック
	exist, err := isExistUserID(p.UserID)
	if err != nil {
		return err
	}
	if exist == false {
		messages = append(messages, "このユーザは未登録または削除済であるため, 投稿出来ません")
	}

	// 投稿メッセージの文字数チェック
	if n := utf8.RuneCountInString(p.Message); n < 1 || 140 < n {
		messages = append(messages, "投稿は1文字以上, 140字以内で行ってください")
	}
	return nil
}

// Entry はDBへ投稿情報を新規登録するメソッド
func (p *Post) Entry() error {
	// コネクション取得
	db, err := DBConnection()
	if err != nil {
		return err
	}

	// プリペアードステートメント生成
	stmt, err := db.Prepare("INSERT INTO posts(user_id, message, created_at) VALUES(?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// 投稿時刻登録
	p.CreatedAt = time.Now()
	// クエリ発行
	result, err := stmt.Exec(p.UserID, p.Message, p.CreatedAt)
	if err != nil {
		return err
	}
	// 登録したIDを構造体へ入れてやる
	insertID, err := result.LastInsertId()
	if err != nil {
		return err
	}
	p.ID = insertID

	return nil
}

// Sweets はuserIDのタイムラインに表示されるSweetを取得する
func Sweets(userID int64, limit int, offset int) ([]Post, error) {
	// コネクション取得
	db, err := DBConnection()
	if err != nil {
		return nil, err
	}
	// SQL発行
	rows, err := db.Query(`
		SELECT
			p.id,
			p.user_id,
			u.name,
			p.message,
			p.created_at
		FROM
			posts p
		INNER JOIN
			users u
		ON
			p.user_id = u.id
		WHERE
			p.user_id = ?
		ORDER BY
			p.created_at desc
		LIMIT ?
		OFFSET ?
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := make([]Post, 0)
	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.UserID, &p.UserName, &p.Message, &p.CreatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, nil
}
