package main

import (
	"database/sql"
	_ "github.com/lib/pq"
)

import (
	"fmt"
	"gopkg.in/gographics/imagick.v3/imagick"
	"io"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
)

type Message struct {
	Id    int64  `json:"id,omitempty"`
	Sid   string `json:"sid"`
	Phone string `json:"phone"`
	Body  string `json:"body"`
	Url   string `json:"url"`
}

var db *sql.DB

func createUser(phone string) {
	sqlStatement := `INSERT INTO users (phone) VALUES ($1) ON CONFLICT DO NOTHING`
	log.Printf("Creating user %s...", phone)
	_, err := db.Exec(sqlStatement, phone)
	if err != nil {panic(err)}
}
func createMessage(phone string, sid string, body string, url string) {
	sqlStatement := `INSERT INTO messages (phone, sid, body, url) VALUES ($1, $2, $3, $4)`
	_, err := db.Exec(sqlStatement, phone, sid, body, url)
	if err != nil {panic(err)}
}

// POST
func handleSms(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	err  := r.ParseForm()
	form := r.Form
	if err != nil {panic(err)}

	smsSid      := form.Get("SmsSid")
	phone       := form.Get("From")[1:]
	createUser(phone)
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
	if err != nil {return err}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {return err}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {return err}

	return nil
}

func queryMessages(phone string) []Message {
	rows, err := db.Query(`SELECT * FROM messages WHERE phone = $1`, phone)
	if err != nil {panic(err)}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		err := rows.Scan(&m.Sid, &m.Phone, &m.Body, &m.Url, &m.Id)
		if err != nil {panic(err)}
		messages = append(messages, m)
	}
	return messages
}

// GET
func provideImage(w http.ResponseWriter, r *http.Request) {
	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	query := r.URL.Query()
	sid   := query.Get("sid")

	var phone string
	db.QueryRow(`SELECT phone FROM messages WHERE sid = $1`, sid).Scan(&phone)

	err := mw.ReadImage(fmt.Sprintf("img/%s/%s.jpg", phone, sid))
	if err != nil {panic(err)}

	width       := float64(mw.GetImageWidth())
	height      := float64(mw.GetImageHeight())
	aspectRatio := width / height

	if aspectRatio > 1 {
		width  = float64(320)
		height = width / aspectRatio
	} else {
		height = float64(320)
		width  = height * aspectRatio
	}

	err = mw.ResizeImage(uint(width), uint(height), imagick.FILTER_LANCZOS)
	if err != nil {panic(err)}
	err = mw.SetImageCompressionQuality(75)
	if err != nil {panic(err)}
	w.Write(mw.GetImageBlob())
}

// GET
func provideMessages(w http.ResponseWriter, r *http.Request) {
	phone    := r.URL.Query().Get("phone")
	messages := queryMessages(phone)

	jData, err := json.Marshal(messages)
	if err != nil {panic(err)}
	w.Header().Set("Content-Type", "application/json")
	w.Write(jData)
}

func main() {
	connStr := "user=postgres dbname=mother sslmode=disable"
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {panic(err)}
	defer db.Close()

	imagick.Initialize()
	defer imagick.Terminate()

	http.HandleFunc("/sms", handleSms)
	http.HandleFunc("/image", provideImage)
	http.HandleFunc("/messages", provideMessages)
	http.ListenAndServe(":8080", nil)
}
