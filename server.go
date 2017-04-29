package main

import (
	_ "github.com/mattn/go-sqlite3"
	"log"
	"database/sql"
	"net"
	"bufio"
	"strings"
	"net/url"
	"errors"
	"fmt"
	"strconv"
)

type HTTPRequest struct {
	reqtype string
	url     *url.URL
}

type Error struct {
	msg  string
	code int
}

type DBRow struct {
	id      int
	hash    string
	longurl string
}

const table_name string = "url_list"

var base62Chars = []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t",
						   "u", "v", "w", "x", "y", "z", "A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q",
						   "R", "S", "T", "U", "V", "W", "X", "Y", "Z", "0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}

var rowNumber int

func main() {
	// Set up the Schema
	// Creates a database if one does not exists
	db, err := SetUpSchema()
	if err != nil {
		log.Fatalf("Error Occurred: %s", err)
	}
	defer db.Close()

	rowNumber, err = GetLastID(db)
	if err != nil {
		log.Fatalf("Error in fetching last id; %s", err)
	}
	// Start listening on PORT 5001
	ln, err := net.Listen("tcp", ":5001")
	if err != nil {
		log.Fatalf("Failed to listen: %s", err)
	}
	// Keep listening for requests and handling them with a go routine.
	for {
		if conn, err := ln.Accept(); err == nil {
			go handleConnection(conn, db)
		}
	}

}

func handleConnection(conn net.Conn, db *sql.DB) {
	defer conn.Close()

	// Create a read buffer
	reader := bufio.NewReader(conn)
	// Read the first line.
	// It's the only thing important here.
	// So don't even bother reading anything else
	if reqData, err := reader.ReadBytes('\n'); err == nil {
		// Parse the first line and get useful stuff
		req, err := ParseHTTPRequest(string(reqData))
		if err != nil {
			conn.Write([]byte(err.msg))
			return
		}
		//Get the value of url key in query string
		submitURL := req.url.Query().Get("url")

		if len(submitURL) > 0 {

			// This part gets executed when submitting a new url
			shorturl, err := SubmitURL(submitURL, db)
			if err != nil {
				SetHeaders(conn, "", "500")
				conn.Write([]byte("Error Occurred, Please try again in a moment\n"))
			} else {
				SetHeaders(conn, "", "200")
				conn.Write([]byte(shorturl + "\n"))
			}
		} else if req.url.Path == "/" {
			// When Someone Visits Home Page

			SetHeaders(conn, "", "200")
			conn.Write([]byte("Welcome to URL Shortner Service."))

		} else {

			// This gets executed when someone visits a shortned url
			// Remove '/' from url
			hash := strings.Replace(req.url.Path, "/", "", 1)

			longurl, err := FindAndRedirect(hash, db)
			if err != nil {
				log.Printf("Error: %s", err)
				SetHeaders(conn, "", "404")
				conn.Write([]byte("Error 404: Not Found"))
				return
			}
			SetHeaders(conn, longurl, "301")
			conn.Write([]byte(longurl + "\n"))
		}
	}
}

func ParseHTTPRequest(reqString string) (*HTTPRequest, *Error) {
	//	Only the first line of request is useful here.
	reqData := strings.Split(reqString, " ")
	if len(reqData) >= 2 {
		reqType := reqData[0]
		reqURL := reqData[1]
		URL, err := url.Parse(reqURL)
		if err != nil {
			return nil, &Error{code: 400, msg: "Bad Request"}
		}
		return &HTTPRequest{reqType, URL}, nil
	}
	return nil, &Error{code: 400, msg: "Bad Request"}
}

func InsertURLInDB(hash string, longurl string, db *sql.DB) error {
	//"INSERT INTO Url_list (id, hash, longurl) values (465, "ishan", "test1");"
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
	//fmt.Printf("%d Row(s) inserted\n", rowsAffected)
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

func SubmitURL(longurl string, db *sql.DB) (string, error) {
	Hash := GenerateShortHash(rowNumber)
	rowNumber++
	fmt.Println(rowNumber)
	error := InsertURLInDB(Hash, longurl, db)
	if error != nil {

	}
	shorturl := "http://localhost:5000/" + Hash
	return shorturl, nil
}

func FindAndRedirect(hash string, db *sql.DB) (string, error) {
	// Little hack to prevent from noobish SQL injection.
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
		return "", errors.New("Not Found")
	}
	rows.Scan(&dbrow.longurl)

	return dbrow.longurl, nil
}

func SetHeaders(conn net.Conn, longurl string, status string) {
	// Set the first line of response
	// Contains HTTP Version, Status Code and a Message.
	switch status {
	case "301":
		conn.Write([]byte("HTTP/1.1 301 Moved Permanently\n"))
		conn.Write([]byte("Location: " + longurl + "\n"))
	case "400":
		conn.Write([]byte("HTTP/1.1 400 Bad Request\n"))
	case "404":
		conn.Write([]byte("HTTP/1.1 404 Not Found\n"))
	case "200":
		conn.Write([]byte("HTTP/1.1 200 OK\n"))
	case "500":
		conn.Write([]byte("HTTP/1.1 500 Internal Server Error\n"))
	}

	conn.Write([]byte("Connection: close\n"))
	// Add a newline, Anything written after this is considered to be the body of response.
	conn.Write([]byte("\n"))
}
