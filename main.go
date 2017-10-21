package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// XXX: unused
const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

// Document is the structure of the document listing
type Document struct {
	ID      string     `db:"id" json:"id"`
	Title   string     `db:"title" json:"title"`
	Body    string     `db:"body" json:"body,omitempty"`
	Created time.Time  `db:"created" json:"created"`
	Updated *time.Time `db:"updated" json:"updated,omitempty"`
}

// DocumentChanges is the structure to catch each change in the document
type DocumentChanges struct {
	DocID     string     `db:"document_id"`
	BodyState string     `db:"body_state"`
	Update    *time.Time `db:"updated"`
}

var (
	// Postgres
	db *sqlx.DB
)

func init() {
	// Seed for randHash
	rand.Seed(time.Now().UnixNano())
}

// XXX: unused
func randHash(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// wsHandler is thewebsocket endpoint
func wsHandler(w http.ResponseWriter, r *http.Request) {
	paths := strings.Split(r.URL.Path, "/")
	id := paths[len(paths)-1]
	log.Println("HEllo", id)
	// check to see if ws endpoint for doc exists, if not creats one
	if hubMap[id] == nil {
		hubMap[id] = newHub(id)
		go hubMap[id].run()
	}

	serveWS(hubMap[id], w, r)
}

// create is a POST only endpoint to create the document
func create(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
		return
	}

	r.ParseForm()

	title := r.Form["title"][0]
	if title == "" {
		log.Println("empty")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var id string
	err := db.Get(&id, `INSERT INTO documents(title) VALUES($1) RETURNING id`, title)
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	log.Println("redirect")
	http.Redirect(w, r, "/documents/"+id, http.StatusSeeOther)
}

// edit is the endpoint to find the document and open the document editor view
func edit(w http.ResponseWriter, r *http.Request) {
	paths := strings.Split(r.URL.Path, "/")
	if paths[len(paths)-1] == "documents" {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	id := paths[len(paths)-1]

	var doc Document
	err := db.Get(&doc, `SELECT * FROM documents WHERE id=$1`, id)
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
	http.ServeFile(w, r, "./public/document.html")
}

func docBodyHandler(w http.ResponseWriter, r *http.Request) {
	var doc Document
	paths := strings.Split(r.URL.Path, "/")
	err := getDoc(&doc, paths[len(paths)-1])
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	err = json.NewEncoder(w).Encode(&doc)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func main() {
	db = sqlx.MustConnect("postgres", "postgres://dbuser@127.0.0.1:5432/doc_colab?sslmode=disable")

	serv := http.Server{
		Addr: "127.0.0.1:3030",
	}

	hubMap = make(map[string]*Hub)

	http.HandleFunc("/docbody/", docBodyHandler)
	http.HandleFunc("/editor/", wsHandler)
	http.HandleFunc("/create", create)
	http.HandleFunc("/documents/", edit)
	http.Handle("/", http.FileServer(http.Dir("./public")))

	log.Printf("Listening on %s\n", serv.Addr)
	err := serv.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
