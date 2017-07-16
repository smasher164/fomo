package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"log"
	"net/http"
	"os"
	"unsafe"

	"github.com/cockroachdb/cockroach-go/crdb"
	"github.com/gtank/cryptopasta"
	fb "github.com/huandu/facebook"
	_ "github.com/lib/pq"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
)

var port, root, authRoot, fbID, fbSecret, acSecret string

func init() {
	flag.StringVar(&port, "port", ":8084", "http port")
	flag.StringVar(&root, "root", "http://localhost:8083", "root url for the site")
	flag.StringVar(&authRoot, "authRoot", "http://localhost:8084/auth", "root url for the auth server")
	flag.StringVar(&fbID, "fbID", "", "client id for facebook oauth2 client")
	flag.StringVar(&fbSecret, "fbSec", "", "client secret for facebook oauth2 client")
	flag.StringVar(&acSecret, "acSec", "", "encryption key for access token")
}

type authserver struct {
	app       *fb.App
	stateconf map[string]*oauth2.Config
	db        *sql.DB
	acSecret  *[32]byte
	logger    *log.Logger
}

func random() string {
	b := make([]byte, 16)
	rand.Read(b)
	s := base64.RawURLEncoding.EncodeToString(b)
	return s
}

// /auth
func (as *authserver) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	method := req.FormValue("method")
	_, redirect := req.URL.Query()["redirect"]
	if method == "facebook" {
		if redirect {
			as.facebookRedirect(rw, req)
		} else {
			as.facebook(rw, req)
		}
	}
}

// /auth?method=facbeook
func (as *authserver) facebook(rw http.ResponseWriter, req *http.Request) {
	conf := &oauth2.Config{
		RedirectURL:  authRoot + "?method=facebook&redirect",
		ClientID:     fbID,
		ClientSecret: fbSecret,
		Scopes:       []string{"public_profile", "user_friends"},
		Endpoint:     facebook.Endpoint,
	}
	state := random()
	authURL := conf.AuthCodeURL(state)
	as.stateconf[state] = conf
	http.Redirect(rw, req, authURL, http.StatusTemporaryRedirect)
}

func createFacebookUser(first, last, fb_id interface{}, user_id *int64) func(*sql.Tx) error {
	return func(tx *sql.Tx) error {
		row := tx.QueryRow("INSERT INTO users (first_name, last_name) VALUES ($1, $2) RETURNING user_id;",
			first,
			last)
		if err := row.Scan(user_id); err != nil {
			return err
		}
		res, err := tx.Exec("INSERT INTO auth (user_id, fb_id, method) VALUES ($1, $2, $3);",
			*user_id,
			fb_id,
			"facebook")
		if err != nil {
			return err
		}
		if n, err := res.RowsAffected(); err != nil || n != 1 {
			return err
		}
		return nil
	}
}

func (as *authserver) facebookRedirect(rw http.ResponseWriter, req *http.Request) {
	state := req.FormValue("state")
	conf, ok := as.stateconf[state]
	delete(as.stateconf, state)
	if !ok {
		as.logger.Println("state variable invalid in redirect")
		http.Redirect(rw, req, root, http.StatusTemporaryRedirect)
		return
	}
	code := req.FormValue("code")
	token, err := conf.Exchange(oauth2.NoContext, code)
	if err != nil {
		as.logger.Println(err)
		http.Redirect(rw, req, root, http.StatusTemporaryRedirect)
		return
	}

	// Fetch facebook user id from app using access token.
	session := as.app.Session(token.AccessToken)
	session.HttpClient = conf.Client(oauth2.NoContext, token)
	session.EnableAppsecretProof(true)
	res, err := session.Get("/me", fb.Params{
		"fields": []string{"first_name", "last_name"},
	})
	if err != nil {
		// probably should log this
		as.logger.Println(err)
		http.Redirect(rw, req, root, http.StatusTemporaryRedirect)
		return
	}
	var user_id int64
	fb_id, first, last := res["id"], res["first_name"], res["last_name"]

	// Use it to look up fb user's associated fomo user_id in database.
	row := as.db.QueryRow("SELECT user_id FROM auth WHERE fb_id = $1", fb_id)

	// if the user doesn't exist, create one
	if err := row.Scan(&user_id); err != nil {
		err = crdb.ExecuteTx(as.db, createFacebookUser(first, last, fb_id, &user_id))
		if err != nil {
			// probably should log this
			as.logger.Println(err)
			http.Redirect(rw, req, root, http.StatusTemporaryRedirect)
			return
		}
	}

	// Encrypt access token with secret key.
	ciphertext, err := cryptopasta.Encrypt([]byte(token.AccessToken), as.acSecret)
	if err != nil {
		// probably should log this
		as.logger.Println(err)
		http.Redirect(rw, req, root, http.StatusTemporaryRedirect)
		return
	}

	// convert ciphertext to hex for easy comparison and client-side communication
	ctHex := hex.EncodeToString(ciphertext)

	// Store encrypted token in auth table associated with the user_id.
	result, err := as.db.Exec("UPDATE auth SET access_token = $1 WHERE user_id = $2;",
		ctHex,
		user_id)
	if err != nil {
		// probably should log this
		as.logger.Println(err)
		http.Redirect(rw, req, root, http.StatusTemporaryRedirect)
		return
	}
	if n, err := result.RowsAffected(); err != nil || n != 1 {
		// probably should log this
		as.logger.Printf("err = %v, n = %v\n", err, n)
		http.Redirect(rw, req, root, http.StatusTemporaryRedirect)
		return
	}

	// Set client cookie to the encrypted token.
	c := sessionCookie(ctHex)
	http.SetCookie(rw, c)

	// Redirect to root.
	http.Redirect(rw, req, root, http.StatusTemporaryRedirect)
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

	as := &authserver{
		app:       fb.New(fbID, fbSecret),
		stateconf: make(map[string]*oauth2.Config),
		db:        db,
		acSecret:  bsec,
		logger:    log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile),
	}

	http.Handle("/auth", as)
	http.ListenAndServe(port, nil)
}
