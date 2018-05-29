package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"./models"

	"github.com/spf13/viper"
	"github.com/patrickmn/go-cache"
	"time"
)


var topics map[int]*models.Topic
var comments map[string]*models.Comment
var app_ache  = cache.New(15*time.Minute, 30*time.Minute)

//Сортировка
func SortKeys(p map[string]string) []string {

	keys := make([]string, 0, len(p))

	for key := range p {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys

}

//Высчитываем sig
func GetMd5Hash(p map[string]string, keys []string) string {
	var buffer bytes.Buffer

	for _, v := range keys {

		buffer.WriteString(v)
		buffer.WriteString("=")
		buffer.WriteString(p[v])
	}

	buffer.WriteString(viper.GetString("okSessionSecretKey"))

	h := md5.New()
	io.WriteString(h, buffer.String())
	return hex.EncodeToString(h.Sum(nil))

}

//Формирум Query
func makeRequest(p map[string]string, keys []string, sig string) *http.Request {

	req, _ := http.NewRequest("GET", viper.GetString("okHost"), nil)
	values := req.URL.Query()

	for _, v := range keys {
		values.Add(v, p[v])
	}

	values.Add("sig", sig)
	values.Add("access_token", viper.GetString("okAccessToken"))

	req.URL.RawQuery = values.Encode()
	return req
}

func sendRequest(p map[string]string) interface{} {

	params := map[string]string{
		"application_key": viper.GetString("okApplicationKey"),
		"format":          viper.GetString("okFormat"),
		"gid":             viper.GetString("okGid"),
	}

	for k, v := range params {
		p[k] = v
	}

	keys := SortKeys(p)
	sig := GetMd5Hash(p, keys)
	r := makeRequest(p, keys, sig)
	client := &http.Client{}
	response, _ := client.Do(r)
	contentResponse, _ := ioutil.ReadAll(response.Body)
	response.Body.Close()

	var f interface{}
	json.Unmarshal([]byte(contentResponse), &f)

	return f
}

func getTopics() {

	params := map[string]string{
		"method": "group.getStatTopics",
		"fields": "ID,COMMENTS",
		"count":  "24",
	}

	m := sendRequest(params).(map[string]interface{})
	t := m["topics"].([]interface{})

	//s := []int{5, 4, 3, 2, 1}
	//for i := len(s)-1; i >= 0; i-- {
	//	fmt.Println(s[i])
	//}

	for i := len(t) - 1; i >= 0; i-- {

		comments = make(map[string]*models.Comment, 0)
		params := map[string]string{
			"discussionId":   t[i].(map[string]interface{})["id"].(string),
			"discussionType": "GROUP_TOPIC",
			"method":         "discussions.get",
		}

		m2 := sendRequest(params).(map[string]interface{})

		discussion := m2["discussion"].(map[string]interface{})
		likeCount := discussion["like_count"].(float64)
		commentCount := discussion["total_comments_count"].(float64)
		entities := m2["entities"].(map[string]interface{})
		themes := entities["themes"].([]interface{})
		images, ok := themes[0].(map[string]interface{})["images"].([]interface{})
		if ok {

			var topic *models.Topic

			removePart := "Публикуем вакансии вахтовым методом"
			id := themes[0].(map[string]interface{})["id"].(string)
			title := strings.Split(themes[0].(map[string]interface{})["title"].(string), removePart)
			image := images[0].(map[string]interface{})["pic640x480"].(string)

			params := map[string]string{
				"discussionId":   id,
				"discussionType": "GROUP_TOPIC",
				"method":         "discussions.getComments",
				"direction":      "BACKWARD",
			}

			m3 := sendRequest(params).(map[string]interface{})

			t, ok := m3["comments"].([]interface{})

			if ok {
				for _, commentItem := range t {

					idComment := commentItem.(map[string]interface{})["id"].(string)
					textComment := commentItem.(map[string]interface{})["text"].(string)
					dateComment := commentItem.(map[string]interface{})["date"].(string)

					var comment *models.Comment
					comment = models.NewComment(idComment, textComment, dateComment)

					comments[idComment] = comment
				}
			}

			topic = models.NewTopic(id, title[0], image, commentCount, likeCount, comments)
			topics[i] = topic

		}

	}

}

func indexHandler(w http.ResponseWriter, _ *http.Request) {



	_, found := app_ache.Get("arrTopics")

	if !found {
		fmt.Println("Request to API OK")
		getTopics()
		app_ache.Set("arrTopics", topics, cache.DefaultExpiration)
	}

	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		fmt.Fprintln(w, err.Error())
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	t.ExecuteTemplate(w, "index", topics)
}

func main() {

	viper.SetConfigName("config")
	viper.AddConfigPath("./config")

	err := viper.ReadInConfig()

	if err != nil { // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	topics = make(map[int]*models.Topic, 0)
 //fmt.Println(topics)

	http.HandleFunc("/", indexHandler)

	http.ListenAndServe(":5050", nil)
}
