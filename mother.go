package main

import (
	"fmt"
	"gopkg.in/gographics/imagick.v3/imagick"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
)

// POST
func handleSms(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	err  := r.ParseForm()
	form := r.Form
	if (err != nil) {
		panic(err)
	}

	smsSid      := form.Get("SmsSid")
	from        := form.Get("From")[1:]
	numMedia, _ := strconv.Atoi(form.Get("NumMedia"))
	for i:= 0; i <= numMedia - 1; i++ {
		cacheImage(from, smsSid, form.Get(fmt.Sprintf("MediaUrl%d", i)))
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
func provideImg(w http.ResponseWriter, r *http.Request) {
	imagick.Initialize()
	defer imagick.Terminate()

	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	query  := r.URL.Query()
	phone  := query.Get("phone")
	sid    := query.Get("sid")
	width  := query.Get("width")
	height := query.Get("height")


}

func main() {
	http.HandleFunc("/sms", handleSms)
	http.HandleFunc("/img", provideImg)
	http.ListenAndServe(":8080", nil)
}
