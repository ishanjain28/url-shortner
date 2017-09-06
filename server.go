package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	"github.com/mattn/go-sqlite3"
)

type DBRow struct {
	id      int
	hash    string
	longurl string
}

const table_name string = "url_list"

var base62Chars = []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t",
	"u", "v", "w", "x", "y", "z", "A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q",
	"R", "S", "T", "U", "V", "W", "X", "Y", "Z", "0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}

var PORT = os.Getenv("PORT")
var WEBSITE_ADDR = os.Getenv("HOST")

var rowNumber int
var mutex = &sync.Mutex{}

func init() {
	if PORT == "" {
		log.Fatalln("$PORT not set")
	}

	if WEBSITE_ADDR == "" {
		log.Fatalln("$HOST not set")
	}
}

func main() {

	db, err := SetUpSchema()
	if err != nil {
		log.Fatalln(err)
	}
	LastID, err := GetLastID(db)
	rowNumber = LastID + 1

	router := mux.NewRouter()
	router.HandleFunc("/{shorturl}", func(w http.ResponseWriter, h *http.Request) {

		urlParams := mux.Vars(h)
		mutex.Lock()
		longurl, err := FindURLInDB(urlParams["shorturl"], db)
		mutex.Unlock()
		if err != nil {
			if err.Error() == "404" {
				fmt.Fprintf(w, `{"error": 1,"message": "URL Not Stored on server" }`)
			} else {
				fmt.Fprintf(w, "Error 500: Internal Server Error")
				log.Panic(err.Error())
			}
		} else {
			http.Redirect(w, h, longurl, 301)
		}
	})

	router.HandleFunc("/", func(w http.ResponseWriter, h *http.Request) {
		urlToAdd := h.URL.Query().Get("url")
		if urlToAdd != "" {

			hash := GenerateShortHash(rowNumber)

			mutex.Lock()
			err := InsertURLInDB(hash, urlToAdd, db)
			mutex.Unlock()
			if err != nil {
				fmt.Println(err.Error())
			}
			rowNumber++
			fmt.Fprintf(w, `{"error": 0, "shorturl":"http://`+WEBSITE_ADDR+"/"+hash+"\"}")
		} else {
			fmt.Fprintf(w, `<html><head><title>URL Shortner Microservice</title></head><body><h2>To create a shorturl of any URL, Send a GET request to https://fcc-shorten-urls.herokuapp.com/?url=<url>. You'll receive a JSON in response with the shorturl in it.</h2> <a href="https://github.com/ishanjain28/url-shortner">Github</a></body></html> `)
		}
	})

	http.ListenAndServe(fmt.Sprintf(":%s", PORT), router)
}

func SetUpSchema() (dbInstance *sql.DB, error error) {
	db, err := sql.Open("sqlite3", path.Join("./urls.db")+"?cache=shared&mode=rwc")
	if err != nil {
		log.Fatalf("Error Occurred in creating Schema: %s", err)
	}

	createTable := `create table ` + table_name + ` (id INTEGER PRIMARY KEY, hash varchar(14), longurl varchar(200))`
	_, err = db.Exec(createTable)

	if error, ok := err.(sqlite3.Error); ok {
		log.Println(error)
		return db, nil
	}
	return db, nil
}

func InsertURLInDB(hash string, longurl string, db *sql.DB) error {

	if strings.Index(longurl, "http://") == -1 || strings.Index(longurl, "https://") == -1 {
		longurl = "http://" + longurl
	}
	//"INSERT INTO url_list (id, hash, longurl) values (465, "ishan", "test1");"
	query := "INSERT INTO " + table_name + "(id, hash, longurl) values(" + strconv.Itoa(rowNumber) + ", \"" + hash + "\", \"" + longurl + "\");"
	queryResult, err := db.Exec(query)
	if err != nil {
		log.Fatalln(err)
		return err
	}
	_, err = queryResult.RowsAffected()
	if err != nil {
		log.Printf("Errored: %s", err)
		return err
	}
	return nil
}

func GetLastID(db *sql.DB) (int, error) {
	query := "SELECT id FROM " + table_name + " ORDER BY id DESC LIMIT 1;"
	rows, err := db.Query(query)
	if err != nil {
		log.Fatalln(err)
		return 0, err
	}
	defer rows.Close()
	row := DBRow{}
	for rows.Next() {
		err := rows.Scan(&row.id)
		if err != nil {
			log.Fatal(err)
		}
	}
	return row.id, nil
}

func GenerateShortHash(LastID int) string {
	rowNo := LastID + 1
	hashInNumbers := []int{}
	hash := []string{}
	for rowNo > 0 {
		remainder := rowNo % 62
		hashInNumbers = append(hashInNumbers, remainder)
		rowNo = rowNo / 62
	}
	for i := len(hashInNumbers) - 1; i >= 0; i-- {
		hash = append(hash, base62Chars[hashInNumbers[i]])
	}
	return strings.Join(hash[:], "")
}

func FindURLInDB(hash string, db *sql.DB) (string, error) {

	hash = strings.Split(hash, " ")[0]

	query := `SELECT longurl from ` + table_name + ` where hash="` + hash + "\""
	rows, err := db.Query(query)
	if err != nil {
		log.Fatalf("Errored: %s", err)
		return "", err
	}
	defer rows.Close()
	dbrow := DBRow{}

	a := rows.Next()
	if !a {
		return "", errors.New("404")
	}
	rows.Scan(&dbrow.longurl)

	return dbrow.longurl, nil
}
