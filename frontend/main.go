package main

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"strings"
	"unsafe"

	_ "github.com/lib/pq"
)

var port, acSecret, cwd string

func init() {
	flag.StringVar(&cwd, "cwd", "", "current working directory")
	flag.StringVar(&port, "port", ":8083", "http port")
	flag.StringVar(&acSecret, "acSec", "", "encryption key for access token")
}

type frontend struct {
	db       *sql.DB
	acSecret *[32]byte
}

func logFn(errs ...error) {
	for _, e := range errs {
		fmt.Println(e)
	}
	os.Exit(1)
}

func (fr *frontend) validateCookie(req *http.Request) error {
	c, err := req.Cookie("SessionID")
	if err != nil {
		// cookie not present
		return err
	}
	row := fr.db.QueryRow("SELECT user_id, fb_id, method FROM auth WHERE access_token = $1", c.Value)
	var user_id int64
	var fb_id, method string
	if err := row.Scan(&user_id, &fb_id, &method); err != nil {
		// not authenticated.
		return err
	}
	// want to return the user info?
	return nil
}

// type decodeHandler func(*json.Decoder, http.ResponseWriter) error

func (fr *frontend) handleDec(typ string, dec *json.Decoder, rw http.ResponseWriter) error {
	switch typ {
	case "create_event":
		var event struct {
			Title string
			Desc  string
			// change these to Time later
			Date string
			Time string
		}
		if err := dec.Decode(&event); err != nil {
			return err
		}
		fmt.Println(event)
		rw.Write([]byte("Success"))
	}
	return nil
}

func (fr *frontend) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if err := fr.validateCookie(req); err != nil {
		template.Must(template.ParseFiles(cwd+"src/landing.gohtml")).Execute(rw, nil)
		return
	}

	// valid user
	if req.Method == "GET" {
		template.Must(template.ParseFiles(cwd+"src/loggedin.gohtml")).Execute(rw, nil)
		return
	} else if req.Method == "POST" {
		if req.Header.Get("Content-Type") == "application/json; charset=UTF-8" {
			dec := json.NewDecoder(req.Body)
			typ, dec := typeof(dec)
			fr.handleDec(typ, dec, rw)
		}
	}

	// // IF I NEED TO FETCH FROM FACEBOOK.
	// ciphertext, err := hex.DecodeString(c.Value)
	// if err != nil {
	// 	// bad access token
	// 	template.Must(template.ParseFiles(cwd + "src/landing.gohtml")).Execute(rw, nil)
	// 	return
	// }
	// access_token, err := cryptopasta.Decrypt(ciphertext, fr.acSecret)
	// if err != nil {
	// 	// bad ciphertext
	// 	template.Must(template.ParseFiles(cwd + "src/landing.gohtml")).Execute(rw, nil)
	// 	return
	// }
	// session := fr.app.Session(string(access_token))
	// session.EnableAppsecretProof(true)
}

func typeof(dec *json.Decoder) (string, *json.Decoder) {
	// type is at beginning
	strTok := ""
	for i := 0; i < 2; i++ {
		t, err := dec.Token()
		if err != nil {
			break
		}
		if t == "type" {
			t, err = dec.Token()
			if err != nil {
				break
			}
			strTok = fmt.Sprint(t)
		}
	}
	// Modify decoder to parse rest of buffer
	r := dec.Buffered()
	r.Read([]byte{1})
	r = io.MultiReader(strings.NewReader("{"), r)
	dec = json.NewDecoder(r)
	return strTok, dec
}

func main() {
	flag.Parse()

	sec, err := hex.DecodeString(acSecret)
	if err != nil {
		panic(err)
	}
	bsec := (*[32]byte)(unsafe.Pointer(&sec[0]))

	db, err := sql.Open("postgres", "postgresql://akhil@localhost:26257/fomo?sslmode=disable")
	if err != nil {
		panic(err)
	}

	fr := &frontend{
		db:       db,
		acSecret: bsec,
	}

	http.Handle("/", fr)
	http.Handle("/src/", http.StripPrefix("/src/", http.FileServer(http.Dir(cwd+"src"))))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(cwd+"static"))))
	http.ListenAndServe(port, nil)
}
