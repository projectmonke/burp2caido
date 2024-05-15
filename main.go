package main

/*
BurpToCaido v1.0 by monke
---
This is a utility to convert HTTP history from Burpsuite to Caido.
Burpsuite HTTP history is exported in XML format. This is formatted and
inserted into Caido's SQLite databases.

Usage:
- Run the binary, specifying the input file and the location of Caido's projects.
./burptocaido --burpsuite <path to XML file> --caido <path to Caido project folder containing database.caido>
*/

import (
	"database/sql"
	"encoding/base64"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Item struct {
	// An Item is a HTTP request/response pair within Burpsuite's exported XML.
	Time           string `xml:"time"`
	URL            string `xml:"url"`
	Host           string `xml:"host"`
	Port           int    `xml:"port"`
	Protocol       string `xml:"protocol"`
	Method         string `xml:"method"`
	Path           string `xml:"path"`
	Extension      string `xml:"extension"`
	Request        string `xml:"request"`
	Status         int    `xml:"status"`
	ResponseLength int    `xml:"responselength"`
	MimeType       string `xml:"mimetype"`
	Response       string `xml:"response"`
	Comment        string `xml:"comment"`
}

func main() {

	var banner = fmt.Sprintf(`
██████╗ ██╗   ██╗██████╗ ██████╗ ██████╗  ██████╗ █████╗ ██╗██████╗  ██████╗ 
██╔══██╗██║   ██║██╔══██╗██╔══██╗╚════██╗██╔════╝██╔══██╗██║██╔══██╗██╔═══██╗
██████╔╝██║   ██║██████╔╝██████╔╝ █████╔╝██║     ███████║██║██║  ██║██║   ██║
██╔══██╗██║   ██║██╔══██╗██╔═══╝ ██╔═══╝ ██║     ██╔══██║██║██║  ██║██║   ██║
██████╔╝╚██████╔╝██║  ██║██║     ███████╗╚██████╗██║  ██║██║██████╔╝╚██████╔╝ by monke v%s
╚═════╝  ╚═════╝ ╚═╝  ╚═╝╚═╝     ╚══════╝ ╚═════╝╚═╝  ╚═╝╚═╝╚═════╝  ╚═════╝                                                                       
`, "1.0")
	fmt.Println(banner)

	burpsuite := flag.String("burp", "", "Path to Burpsuite XML file")
	caido := flag.String("caido", "", "Path to Caido project path")
	flag.Parse()

	if *burpsuite == "" {
		log.Fatalf("The --burp flag is required.")
		os.Exit(1)
	}

	if *caido == "" {
		log.Fatalf("The --caido flag is required.")
		os.Exit(1)
	}

	fmt.Println("[INFO] Using Caido path: " + *caido)
	fmt.Println("[INFO] Using Burpsuite path: " + *burpsuite)

	db_main_path := *caido + "/database.caido"
	if _, err := os.Stat(db_main_path); os.IsNotExist(err) {
		log.Fatal("Caido main database does not exist.")
	}

	dbCaido, err := sql.Open("sqlite3", db_main_path)
	if err != nil {
		log.Fatalf("Error opening database.caido: %v", err)
	}
	defer dbCaido.Close()

	db_raw_path := *caido + "/database_raw.caido"
	if _, err := os.Stat(db_raw_path); os.IsNotExist(err) {
		log.Fatal("Caido raw database does not exist.")
	}

	dbCaidoRaw, err := sql.Open("sqlite3", db_raw_path)
	if err != nil {
		log.Fatalf("Error opening database_raw.caido: %v", err)
	}
	defer dbCaidoRaw.Close()

	xmlFile, _ := os.Open(*burpsuite)
	defer xmlFile.Close()

	decoder := xml.NewDecoder(xmlFile)
	for {
		token, _ := decoder.Token()
		if token == nil {
			break
		}

		switch se := token.(type) {
		case xml.StartElement:
			if se.Name.Local == "item" {
				var item Item
				decoder.DecodeElement(&item, &se)
				insertData(dbCaido, dbCaidoRaw, item)
			}
		}
	}
	fmt.Println("\033[32m[INFO] Updated Caido databases successfully.\033[0m")
}

func insertData(dbCaido, dbCaidoRaw *sql.DB, item Item) {
	// Used to insert the cleaned data into Caido's databases.
	requestData, _ := base64.StdEncoding.DecodeString(item.Request)
	responseData, _ := base64.StdEncoding.DecodeString(item.Response)

	layout := "Mon Jan 02 15:04:05 MST 2006"
	parsedTime, err := time.Parse(layout, item.Time)
	if err != nil {
		log.Fatalf("Failed to parse datetime: %v", err)
	}
	timestamp := parsedTime.UnixNano() / int64(time.Millisecond)

	// database_raw.caido
	txRaw, err := dbCaidoRaw.Begin()
	if err != nil {
		log.Fatal(err)
	}

	res, err := txRaw.Exec("INSERT INTO requests_raw (data, source, alteration) VALUES (?, 'intercept', 'none')", requestData)
	if err != nil {
		txRaw.Rollback()
		log.Fatal(err)
	}
	rawRequestID, err := res.LastInsertId()
	if err != nil {
		txRaw.Rollback()
		log.Fatal(err)
	}

	res, err = txRaw.Exec("INSERT INTO responses_raw (data, source, alteration) VALUES (?, 'intercept', 'none')", responseData)
	if err != nil {
		txRaw.Rollback()
		log.Fatal(err)
	}
	rawResponseID, err := res.LastInsertId()
	if err != nil {
		txRaw.Rollback()
		log.Fatal(err)
	}
	txRaw.Commit()

	// database.caido
	tx, err := dbCaido.Begin()
	if err != nil {
		log.Fatal(err)
	}

	_, err = tx.Exec("INSERT INTO responses (status_code, raw_id, length, alteration, edited, roundtrip_time, created_at) VALUES (?, ?, ?, 'none', 0, 0, ?)",
		item.Status, rawResponseID, item.ResponseLength, timestamp)
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	responseID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	requestID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}
	_, err = tx.Exec("INSERT INTO requests_metadata (id) VALUES (?)", requestID)
	if err != nil {
		txRaw.Rollback()
		log.Fatal(err)
	}

	_, err = tx.Exec("INSERT INTO requests (host, method, path, length, port, is_tls, raw_id, query, response_id, source, created_at, metadata_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'intercept', ?, ?)",
		item.Host, item.Method, item.Path, len(requestData), item.Port, item.Protocol == "https", rawRequestID, "", responseID, timestamp, rawRequestID)
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	intercept_id, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	_, err = tx.Exec("INSERT INTO intercept_entries (id, request_id) VALUES (?, ?)",
		intercept_id, intercept_id)
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	tx.Commit()
}
