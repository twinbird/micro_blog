package main

import (
	"time"
)

// FollowerForTemplate はフォロアー検索画面表示制御用の構造体
type FollowerForTemplate struct {
	Followers []Follower
}

// Follower は画面表示用のフォロアー情報
type Follower struct {
	FollowerID int64
	UserID     int64
	Name       string
	Following  bool
	Messages   []string
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
		if err := rows.Scan(&f.FollowerID, &f.Name, &followStatus); err != nil {
			return nil, err
		}
		if followStatus == "NO_FOLLOWING" {
			f.Following = false
		} else {
			f.Following = true
		}
		f.UserID = uid
		followers = append(followers, f)
	}
	return followers, nil
}

// Validate はFollowerの登録前の入力チェックを行う
func (f *Follower) Validate() error {
	var messages []string

	if exist, err := isExistUserID(f.UserID); err != nil {
		return err
	} else if exist == false {
		messages = append(messages, "このフォロワーのアカウントは既に削除されています")
	}
	f.Messages = messages
	return nil
}

// Entry はFollowerの情報登録を行う
func (f *Follower) Entry() error {
	// コネクション取得
	db, err := DBConnection()
	if err != nil {
		return err
	}

	// プリペアードステートメント生成
	stmt, err := db.Prepare("INSERT INTO followers(user_id, follower_id, created_at) VALUES(?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// クエリ発行
	_, err = stmt.Exec(f.UserID, f.FollowerID, time.Now())
	if err != nil {
		return err
	}

	return nil
}

// Remove はFollowerの情報登録を行う
func (f *Follower) Remove() error {
	// コネクション取得
	db, err := DBConnection()
	if err != nil {
		return err
	}

	// プリペアードステートメント生成
	stmt, err := db.Prepare(`
		DELETE
		FROM
			followers
		WHERE
			user_id = ?
		AND
			follower_id = ?
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	// クエリ発行
	_, err = stmt.Exec(f.UserID, f.FollowerID)
	if err != nil {
		return err
	}

	return nil
}
