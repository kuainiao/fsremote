// fs-import project main.go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/hearts.zhang/xiuxiu"
)

const (
	edit_distance = 1
)

type Terms struct {
	Terms []string `json:"terms,omitempty"`
}

type handler func(w http.ResponseWriter, r *http.Request)

var (
	addr, sego, face, jieba, fuzzy, es_front string
	_medias                                  = make(map[int]*xiuxiu.EsMedia)
	_terms                                   = make(map[string]float64)
)

func init() {
	flag.StringVar(&addr, "addr", ":8082", "listen address")
	flag.StringVar(&face, "face", "[fe80::fabc:12ff:fea2:64a6]:6767", "libface address")
	flag.StringVar(&sego, "sego", "[fe80::fabc:12ff:fea2:64a6]:8081", "sego address")
	flag.StringVar(&fuzzy, "fuzzy", "[fe80::fabc:12ff:fea2:64a6]:8089", "sego address")
	flag.StringVar(&jieba, "jieba", "[fe80::fabc:12ff:fea2:64a6]:8083", "sego address")
	flag.StringVar(&es_front, "front", "172.17.5.29:80", "es front-end address")
	//	_fuzzy.SetDepth(edit_distance)
}

func main() {
	flag.Parse()
	load_medias()

	log.Println("start server")
	http.Handle("/app/select", handler(handle_app_select))             //name=&pkgs=
	http.Handle("/app/match", handler(handle_app_match))               //name=
	http.Handle("/app/es/select", handler(handle_app_es_select))       //name=&pkgs=
	http.Handle("/fsmedia/face/term", handler(handle_face_term))       //t=term&n=
	http.Handle("/sego/seg", handler(handle_sego_seg))                 //text=
	http.Handle("/jieba/seg", handler(handle_jieba_seg))               //text=
	http.Handle("/fsmedia/fuzzy/term", handler(handle_fuzzy_term))     //term=
	http.Handle("/fsmedia/es/term", handler(handle_es_term))           //term=&from=&to=
	http.Handle("/fsmedia/es-dev/term", handler(handle_es_dev_term))   //term=&indice=&from=&to=
	http.Handle("/fsmedia/es-dev/check", handler(handle_es_dev_check)) //term=&indice=&from=&to=
	http.Handle("/img/sogou", handler(handle_img_sogou))               //q=&w=300&h=200
	http.Handle("/img/redirect.jpg", handler(handle_img_redirect))     //q=&w=200&h=400
	http.Handle("/pinyin/slug", handler(handle_pinyin_slug))           //hans=
	http.Handle("/log/", handler(handle_report))                       //*=*
	http.ListenAndServe(addr, nil)
}

//text=
func handle_jieba_seg(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	text := r.FormValue("text")
	terms := jieba_segment(text)
	panic_error(json.NewEncoder(w).Encode(&terms))
}

//text=
func handle_sego_seg(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	text := r.FormValue("text")
	terms := sego_segment(text)
	panic_error(json.NewEncoder(w).Encode(&terms))
}

//term=
func handle_fuzzy_term(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	term := r.FormValue("term")
	x := fuzzy_trim(fuzzy_suggest(term))
	panic_error(json.NewEncoder(w).Encode(map[string]interface{}{"items": x}))
}

//t=term&n=
func handle_face_term(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	term := r.FormValue("t")
	n := atoi(r.FormValue("n"), 16)

	x := face_trim(face_suggest(term, n))

	panic_error(json.NewEncoder(w).Encode(map[string]interface{}{"items": x}))
}

//name=&pkgs=
func handle_app_select(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	name, pkgs := r.FormValue("name"), r.FormValue("pkgs")
	selected := package_select(pkgs, name)
	panic_error(json.NewEncoder(w).Encode(map[string]interface{}{"items": selected}))
}

//name=&pkgs=
func handle_app_es_select(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	name, pkgs := r.FormValue("name"), r.FormValue("pkgs")
	if name == "" {
		name = r.FormValue("tags")
	}
	uri := es_app_select_url(name, pkgs)
	w.Header().Del("Content-Type")
	w.Header().Set("Location", uri)
	w.WriteHeader(http.StatusFound)
}

//name
func handle_app_match(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	name := r.FormValue("name")
	matched := package_name_match(name)
	panic_error(json.NewEncoder(w).Encode(map[string]interface{}{"items": matched}))
}

//term=
func handle_es_term(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	term, from, to := r.FormValue("term"), r.FormValue("from"), r.FormValue("to")
	f, t := atoi(from, 0), atoi(to, 5)

	uri := es_media_url(term, "media", f, t)
	fmt.Println(uri)

	w.Header().Del("Content-Type")
	w.Header().Set("Location", uri)
	w.WriteHeader(http.StatusFound)
}

func atoi(s string, dft int) (v int) {
	v, err := strconv.Atoi(s)
	if err != nil {
		v = dft
	}
	return
}

//term=&indice=&from=&to=
func handle_es_dev_term(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	term, indice, from, to := r.FormValue("term"), r.FormValue("indice"), r.FormValue("from"), r.FormValue("to")
	if indice == "" {
		indice = "media4"
	}
	f, t := atoi(from, 0), atoi(to, 5)

	uri := es_media_url(term, indice, f, t)
	fmt.Println(uri)

	w.Header().Del("Content-Type")
	w.Header().Set("Location", uri)
	w.WriteHeader(http.StatusFound)
}

//term=&indice=&from=&to=
func handle_es_dev_check(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	term, indice, from, to := r.FormValue("term"), r.FormValue("indice"), r.FormValue("from"), r.FormValue("to")
	if indice == "" {
		indice = "media4"
	}
	f, t := atoi(from, 0), atoi(to, 5)

	medias, num := es_medias_simple(term, indice, f, t)
	panic_error(json.NewEncoder(w).Encode(map[string]interface{}{
		"num":  num,
		"data": medias,
	}))
}

//q=&w=300&h=200
func handle_img_sogou(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	q, pw, ph := r.FormValue("q"), r.FormValue("w"), r.FormValue("h")
	url, width, height := sogou_pic(q, atoi(pw, 0), atoi(ph, 0))

	panic_error(json.NewEncoder(w).Encode(map[string]interface{}{
		"uri":    url,
		"width":  width,
		"height": height,
	}))
}

//q=&w=300&h=200
func handle_img_redirect(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	q := r.FormValue("q")
	q, pw, ph := r.FormValue("q"), r.FormValue("w"), r.FormValue("h")
	url, width, height := sogou_pic(q, atoi(pw, 0), atoi(ph, 0))
	w.Header().Del("Content-Type")
	w.Header().Set("Location", url)
	w.Header().Set("X-PIC", strconv.Itoa(width)+"x"+strconv.Itoa(height))
	w.WriteHeader(http.StatusFound)
}

//hans=
func handle_pinyin_slug(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	hans := r.FormValue("hans")
	pinyin := hans_pinyin(hans)
	panic_error(json.NewEncoder(w).Encode(map[string]interface{}{
		"pinyin": pinyin,
	}))
}

func handle_report(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	url := es_log_url(r.URL.Path, r.URL.RawQuery)

	fmt.Println(url)

	req, err := http.NewRequest("POST", url, r.Body)
	panic_error(err)

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	panic_error(err)

	_, err = io.Copy(w, resp.Body)
	panic_error(err)

	defer resp.Body.Close()
}

func (imp handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	defer func() {
		if err := recover(); err != nil {
			http.Error(w, err.(error).Error(), http.StatusInternalServerError)
		}
	}()
	imp(w, r)
}

func panic_error(err error) {
	if err != nil {
		panic(err)
	}
}
