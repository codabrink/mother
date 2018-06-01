package main

import ( // postgres..
	"database/sql"
	_ "github.com/lib/pq"
)

import (
	"fmt"
	"gopkg.in/gographics/imagick.v3/imagick"
	_ "github.com/davecgh/go-spew/spew"
	"io"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
)

type Message struct {
	Id    int64
	Sid   string
	Phone string
	Body  string
	Url   string
}

var db *sql.DB

func createUser(phone string) {
	sqlStatement := `INSERT INTO users (phone) VALUES ($1) ON CONFLICT DO NOTHING`
	log.Printf("Creating user %s...", phone)
	_, err := db.Exec(sqlStatement, phone)
	if err != nil {
		panic(err)
	}
}
func createMessage(phone string, sid string, body string, url string) {
	sqlStatement := `INSERT INTO messages (phone, sid, body, url) VALUES ($1, $2, $3, $4)`
	_, err := db.Exec(sqlStatement, phone, sid, body, url)
	if err != nil {
		panic(err)
	}
}

// POST
func handleSms(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	err  := r.ParseForm()
	form := r.Form
	if (err != nil) {
		panic(err)
	}

	smsSid      := form.Get("SmsSid")
	phone       := form.Get("From")[1:]
	//createUser(phone)
	numMedia, _ := strconv.Atoi(form.Get("NumMedia"))
	for i:= 0; i <= numMedia - 1; i++ {
		url := form.Get(fmt.Sprintf("MediaUrl%d", i))
		cacheImage(phone, smsSid, url)
		log.Println("Creating message...")
		createMessage(phone, smsSid, form.Get("Body"), url)
	}
}

func cacheImage(phone string, sid string, url string) error {
	os.MkdirAll(fmt.Sprintf("img/%s", phone), os.ModePerm)
	out, err := os.Create(fmt.Sprintf("img/%s/%s.jpg", phone, sid))
	if (err != nil) {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if (err != nil) {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if (err != nil) {
		return err
	}

	return nil
}

// GET
func provideImages(w http.ResponseWriter, r *http.Request) {}

// GET
func provideImage(w http.ResponseWriter, r *http.Request) {
	imagick.Initialize()
	defer imagick.Terminate()

	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	//query  := r.URL.Query()
	//phone  := query.Get("phone")
	//sid    := query.Get("sid")
	//width  := query.Get("width")
	//height := query.Get("height")
}

// GET
func provideMessages(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT * FROM messages WHERE phone = $1`, r.URL.Query().Get("phone"))
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var (
		sid   string
		phone string
		body  string
		url   string
		id    int64
	)
	var messages []Message
	for rows.Next() {
		err := rows.Scan(&sid, &phone, &body, &url, &id)
		if err != nil {
			panic(err)
		}
		messages = append(messages, Message{Sid: sid, Phone: phone, Body: body, Url: url, Id: id})
	}

	jData, err := json.Marshal(messages)
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(jData)
}

func main() {
	connStr := "user=postgres dbname=mother sslmode=disable"
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	http.HandleFunc("/sms", handleSms)
	http.HandleFunc("/image", provideImage)
	http.HandleFunc("/images", provideImages)
	http.HandleFunc("/messages", provideMessages)
	http.ListenAndServe(":8080", nil)
}
