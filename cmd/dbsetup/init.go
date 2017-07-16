package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"regexp"

	"github.com/cockroachdb/cockroach-go/crdb"
	_ "github.com/lib/pq"
)

type Config struct {
	Host      string
	AdminPort string
	SQLPort   string
	User      string
	Dbname    string
	Insecure  bool
	db        *sql.DB
}

// creates a node if not exists
// assumes cockroachdb is installed
// for now:
// cockroach start --insecure \
// --host=localhost \
// --http-port=8080 \
// --port=26257 \
// --background
func (c *Config) Start() error {
	lsAdmin := c.AdminPort
	if c.AdminPort == "8080" {
		lsAdmin = "http-alt"
	}
	lsof := exec.Command("lsof", "-c", "cockroach")
	proc, _ := lsof.CombinedOutput()
	adminLive, _ := regexp.Match(lsAdmin, proc)
	sqlLive, _ := regexp.Match(c.SQLPort, proc)
	if adminLive && sqlLive {
		// node exists. return
		return nil
	}
	// if either one of them is not live, we shut down cockroach
	args := []string{"quit"}
	if c.Insecure {
		args = append(args, "--insecure")
	}
	quit := exec.Command("cockroach", args...)
	quit.Run() // stderr is fine

	// create node
	args = []string{"start",
		"--host=" + c.Host,
		"--http-port=" + c.AdminPort,
		"--port=" + c.SQLPort,
		"--background"}
	if c.Insecure {
		args = append(args, "--insecure")
	}
	create := exec.Command("cockroach", args...)
	if err := create.Run(); err != nil {
		return err
	}
	return nil
}

// create user
// cockroach user get maxroach --insecure --format records
func (c *Config) CreateUser() {
	args := []string{"user",
		"set",
		c.User}
	if c.Insecure {
		args = append(args, "--insecure")
	}
	set := exec.Command("cockroach", args...)
	set.Run()
}

// create database if not exists
// grant privileges to user
// set up tables
// WARNING: STRING CONCATENATED QUERIES HERE. DOCUMENT USAGE!!!
func (c *Config) CreateDb() {
	args := []string{"sql",
		"--execute",
		"CREATE DATABASE " + c.Dbname}
	if c.Insecure {
		args = append(args, "--insecure")
	}

	create := exec.Command("cockroach", args...)
	create.Run()
	args = []string{"sql",
		"--execute",
		"GRANT ALL ON DATABASE " + c.Dbname + " TO " + c.User}
	if c.Insecure {
		args = append(args, "--insecure")
	}
	grant := exec.Command("cockroach", args...)
	grant.Run()

	db, err := sql.Open("postgres", "postgresql://"+c.User+"@"+c.Host+":"+c.SQLPort+"/"+c.Dbname+"?sslmode=disable")
	if err != nil {
		fmt.Println("error connecting to the database: ", err)
		os.Exit(1)
	}
	if err := crdb.ExecuteTx(db, c.createTables); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	c.db = db
}

/*
	CREATE TABLE IF NOT EXISTS users (user_id SERIAL, first_name TEXT, last_name TEXT);
	CREATE TABLE IF NOT EXISTS user_to_events (user_id INT PRIMARY KEY, event_id INT, status INT);
	CREATE TABLE IF NOT EXISTS event_to_users (event_id INT PRIMARY KEY, user_id INT, status INT);
	CREATE TABLE IF NOT EXISTS events (event_id INT PRIMARY KEY, description TEXT, start TIMESTAMPTZ, end TIMESTAMPTZ);
	CREATE TABLE IF NOT EXISTS auth (user_id INT PRIMARY KEY, fb_id TEXT, method TEXT, access_token TEXT);
*/
func (c Config) createTables(tx *sql.Tx) error {
	if _, err := tx.Exec("CREATE TABLE IF NOT EXISTS users (user_id SERIAL, first_name TEXT, last_name TEXT);"); err != nil {
		return err
	}
	if _, err := tx.Exec("CREATE TABLE IF NOT EXISTS user_to_events (user_id INT PRIMARY KEY, event_id INT, status INT);"); err != nil {
		return err
	}
	if _, err := tx.Exec("CREATE TABLE IF NOT EXISTS event_to_users (event_id INT PRIMARY KEY, user_id INT, status INT);"); err != nil {
		return err
	}
	if _, err := tx.Exec("CREATE TABLE IF NOT EXISTS events (event_id INT PRIMARY KEY, description TEXT, start TIMESTAMPTZ, end TIMESTAMPTZ);"); err != nil {
		return err
	}
	if _, err := tx.Exec("CREATE TABLE IF NOT EXISTS auth (user_id INT PRIMARY KEY, fb_id TEXT, method TEXT, access_token TEXT);"); err != nil {
		return err
	}
	return nil
}

// Truncate all tables in fomo database.
func (c *Config) WipeTables() {
	_, err := c.db.Exec(`'TRUNCATE TABLE users, user_to_events, event_to_users, events, auth;'`)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
