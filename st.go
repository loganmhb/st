package main

import "encoding/base64"
import "net/http"
import "log"
import "fmt"
import "flag"
import "database/sql"
import "crypto/rand"
import "strings"
import _ "github.com/mattn/go-sqlite3"

func indexPageWithCsrfToken(token string) string {
	template := `<!DOCTYPE html>
<html>
  <head>
	<title>st</title>
  </head>
  <body>
	<p>Add a link</p>
	<form action="/add" method="POST">
	  <div style="display: block">
		<label for="url">URL to shorten</label>
		<input type="text" name="url" id="url>
	  </div>
	  <div style="display:block">
		<label for="name">Link name (path will be /:name)</label>
		<input type="text" name="name" id="name">
	  </div>
	  <input type="hidden" name="csrftoken" value="%s">
	  <input type="submit" value="Add link">
	</form>
  </body>
</html>
`
	return fmt.Sprintf(template, token)
}

func addLink(db *sql.DB, name string, url string) error {
	stmt := "INSERT INTO links (name, url) VALUES (?, ?)"
	_, err := db.Exec(stmt, name, url)
	return err
}

func getLink(db *sql.DB, name string) (string, error) {
	log.Printf("Fetching link %s", name)
	query := "SELECT url FROM links WHERE name = ?"
	rows, err := db.Query(query, name)

	if err != nil {
		return "", err
	}

	defer rows.Close()
	found := rows.Next()
	if !found {
		log.Printf("link not found")
		return "", nil
	}

	var url string

	err = rows.Scan(&url)
	if err != nil {
		return "", err
	}

	return url, nil
}

func initializeDb(db *sql.DB) error {
	stmt := "CREATE TABLE IF NOT EXISTS links (name text not null primary key, url text not null)"
	_, err := db.Exec(stmt)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func generateToken() (string, error) {
	token := make([]byte, 32)
	_, err := rand.Read(token)
	if err != nil {
		return "", err
	} else {
		return base64.StdEncoding.EncodeToString(token), nil
	}
}

func main() {
	var port = flag.Int("port", 8080, "port on which to start the server")
	var dbFile = flag.String("db", "st.sqlite3", "file in which to store the SQLite link db")
	flag.Parse()

	var csrfTokens = make(map[string]bool)

	db, err := sql.Open("sqlite3", *dbFile)
	if err != nil {
		log.Fatal("error opening sqlite db", err)
	}
	defer db.Close()

	err = initializeDb(db)
	if err != nil {
		log.Fatal("error initializing database", err)
	}

	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		log.Printf("Handling request at path %s", req.URL.Path)

		if req.URL.Path == "/favicon.ico" {
			res.WriteHeader(404)
			return
		}

		if req.URL.Path == "/add" {
			if req.Method == "GET" {
				token, err := generateToken()
				if err != nil {
					log.Printf("Error generating CSRF token: %s", err)
					res.WriteHeader(500)
					return
				}
				csrfTokens[token] = true
				res.Header().Set("Content-Type", "text/html")
				res.WriteHeader(200)
				res.Write([]byte(indexPageWithCsrfToken(token)))
				return
			} else if req.Method == "POST" {
				err := req.ParseForm()
				if err != nil {
					log.Printf("Error parsing form values: %s", err)
					res.WriteHeader(400)
					res.Write([]byte("unable to parse form values"))
					return
				}

				linkUrl := req.PostForm.Get("url")
				linkName := req.PostForm.Get("name")
				csrfToken := req.PostForm.Get("csrftoken")

				if csrfTokens[csrfToken] != true {
					log.Printf("csrf token: %s", csrfToken)
					log.Printf("valid tokens: %v", csrfTokens)

					res.WriteHeader(400)
					res.Write([]byte("invalid csrf token"))
					return
				}

				delete(csrfTokens, csrfToken)

				if linkUrl == "" || linkName == "" {
					res.WriteHeader(400)
					res.Write([]byte("must provide both link name and url"))
					return
				}

				log.Printf("adding link %s (%s)", linkName, linkUrl)
				err = addLink(db, linkName, linkUrl)
				if err != nil {
					log.Printf("error adding link: %s", err)
					res.WriteHeader(500)
					res.Write([]byte("error adding link"))
					return
				}

				res.WriteHeader(200)
				res.Write([]byte("added link"))
				return
			} else {
				res.WriteHeader(400)
				res.Write([]byte("unsupported method"))
				return
			}
		}

		log.Printf("fetching link")
		linkName := strings.TrimPrefix(req.URL.Path, "/")
		linkUrl, err := getLink(db, linkName)
		if err != nil {
			log.Printf("error retrieving link: %s", err)
			res.WriteHeader(500)
			res.Write([]byte("error retrieving link"))
			return
		}

		if linkUrl == "" {
			res.WriteHeader(404)
			res.Write([]byte("link not found"))
			return
		}

		log.Printf("Redirecting to %s for link %s", linkUrl, linkName)

		res.Header().Set("Location", linkUrl)
		res.WriteHeader(301)
		return
	})
	log.Printf("Listening on port %d", *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
