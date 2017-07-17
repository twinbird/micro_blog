package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"path/filepath"
)

// アプリケーション設定
var applicationConfig *Config

// Config はプログラムの起動時設定を格納する
type Config struct {
	DBUser     string
	DBPassword string
	DBName     string
}

// 設定ファイルの読み出し
func loadConfig(path string) (*Config, error) {
	p := filepath.Join(path, "config.json")
	bytes, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var config Config
	if err := json.Unmarshal(bytes, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// DBConnection は*sql.DBを返すメソッド
// 他のDBMSを使いたくなったときに複数個所の変更は面倒なので局所化した
func DBConnection() (*sql.DB, error) {
	conStr := fmt.Sprintf("%s:%s@/%s?parseTime=true",
		applicationConfig.DBUser,
		applicationConfig.DBPassword,
		applicationConfig.DBName)
	return sql.Open("mysql", conStr)
}
