// 実行方法
// まず、
// C:\gocode\src\github.com\useiichi\room> go build
//
// 直で実行：
// C:\gocode\src\github.com\useiichi\room> go run server.go
// http://localhost:1323/taka2 にアクセス
//
// Realizeで実行：
// go get -u github.com/oxequa/realize
// C:\gocode\src\github.com\useiichi\room> realize start
// ctl+C
// .realize.yaml
// schema:
// - name: realize-sample
//   path: .
//   # 修正ここから
//   commands:
//     run:
//       status: true
//   # 修正ここまで
//   watcher:
//     paths:
//     - /
// 再度、
// C:\gocode\src\github.com\useiichi\room> realize start --server
//
// ginで実行（ホットリロード（LiveReload））：
// C:\gocode\src\github.com\useiichi\room> gin
// http://localhost:3000/taka2 にアクセス
// ※ gin のインストール方法：
// go get github.com/codegangsta/gin
// ※ cockroackdbのnodePort: 26257を開けておく必要あり。↓
//---
//apiVersion: v1
//kind: Service
//metadata:
//  # This service is meant to be used by clients of the database. It exposes a ClusterIP that will
//  # automatically load balance connections to the different database pods.
//  name: cockroachdb-public
//  labels:
//    app: cockroachdb
//spec:
//  ports:
//  # The main port, served by gRPC, serves Postgres-flavor SQL, internode
//  # traffic and the cli.
//  - port: 26257
//    nodePort: 26257 ←←←☆☆☆☆☆☆
//    targetPort: 26257
//    name: grpc
//  # The secondary port serves the UI as well as health and debug endpoints.
//  - port: 8080
//    targetPort: 8080
//    name: http
//  selector:
//    app: cockroachdb
//  type: NodePort ←←←☆☆☆☆☆☆
//---
package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/upper/db/v4"
	"github.com/upper/db/v4/adapter/cockroachdb"
)

// CREATE DATABASE taka;
// SHOW DATABASES;
// USE taka;
// CREATE TABLE messages (id INT NOT NULL DEFAULT unique_rowid(),userid INT NULL,body STRING NULL,created_at TIMESTAMP NULL,updated_at TIMESTAMP NULL,CONSTRAINT "primary" PRIMARY KEY (id ASC),FAMILY "primary" (id, userid, body, created_at, updated_at));
// CREATE TABLE messages (
// id INT NOT NULL DEFAULT unique_rowid(),
// userid INT NULL,
// body STRING NULL,
// created_at TIMESTAMP NULL,
// updated_at TIMESTAMP NULL,
// CONSTRAINT "primary" PRIMARY KEY (id ASC),
// FAMILY "primary" (id, userid, body, created_at, updated_at)
// );
// CREATE USER uuu WITH PASSWORD 'oohana';
// select * from pg_user;
// GRANT ALL ON DATABASE taka TO uuu;
// SHOW GRANTS ON DATABASE taka;
// GRANT ALL ON TABLE taka.* TO uuu;
// SHOW GRANTS ON TABLE taka.*;
//
// ↓別のAuto Increment方法（これだと、idが484778898812534786のようになる）
//
// CREATE TABLE messages (
// 	 id SERIAL NOT NULL,
// 	 userid INT NULL,
// 	 body STRING NULL,
//	 created_at TIMESTAMP NULL,
//	 updated_at TIMESTAMP NULL,
//	 CONSTRAINT "primary" PRIMARY KEY (id ASC),
//	 FAMILY "primary" (id, userid, body, created_at, updated_at)
//	 );

// The settings variable stores connection details.
var settings cockroachdb.ConnectionURL

// Message is used to represent a single record in the "messages" table.
type Message struct {
	Id        int       `db:"id,omitempty"`
	Userid    int       `db:"userid"`
	Body      string    `db:"body"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func main() {
	e := echo.New()

	name, err := os.Hostname()
	if err != nil {
		e.Logger.Fatal(err)
	}
	fmt.Printf("Hostname: %s\n", name)

	if name == "DESKTOP-B9KGMU7" {
		settings = cockroachdb.ConnectionURL{
			Host:     "localhost",
			Database: "taka",
			User:     "uuu",
			Options: map[string]string{
				// Insecure node.
				"sslmode": "disable",
			},
		}
	} else {
		settings = cockroachdb.ConnectionURL{
			Host:     "cockroachdb-public.default.svc.cluster.local",
			Database: "taka",
			User:     "root",
			//Password: "oohana",
			Options: map[string]string{
				// Secure node.
				"sslrootcert": "/cockroach-certs/ca.crt",
				"sslkey":      "/cockroach-certs/client.root.key",
				"sslcert":     "/cockroach-certs/client.root.crt",
			},
		}
	}

	e.Static("/taka2/assets", "assets")

	e.Use(session.Middleware(sessions.NewCookieStore([]byte("secret"))))
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "method=${method}, uri=${uri}, status=${status}\n",
	}))

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})

	e.Static("/static", "static")

	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"mul": func(a, b int) int { return a * b },
		"div": func(a, b int) int { return a / b },
		"mod": func(a, b int) int { return a % b },
		"nl2br": func(text string) template.HTML {
			return template.HTML(strings.Replace(template.HTMLEscapeString(text), "\n", "<br />", -1))
		},
		"dt":  func(a time.Time) string { return a.Format("2006年01月02日, 15:04:05") },
		"len": func(a []int) int { return len(a) },
	}

	t := &Template{
		templates: template.Must(template.New("calculator").Funcs(funcMap).ParseGlob("public/views/*.html")),
	}
	e.Renderer = t

	// Route level middleware
	track := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sess, _ := session.Get("session", c)
			sess.Options = &sessions.Options{
				Path:     "/",
				MaxAge:   86400 * 7,
				HttpOnly: true,
			}
			var s int
			if sess.Values["user_id"] == nil {
				s = 0
			} else {
				s = sess.Values["user_id"].(int)
			}
			if s == 0 {
				return c.Redirect(302, "/taka2/sessions/new")
			}
			return next(c)
		}
	}

	// sessions/ など追加
	e.GET("/taka2/sessions/new", SessionsNew)
	e.GET("/taka2/sessions/new/", SessionsNew)
	e.GET("/taka2/sessions", SessionsNew)
	e.GET("/taka2/sessions/", SessionsNew)
	e.POST("/taka2/sessions", createSessions)
	e.GET("/taka2/sessions/:id/delete", SessionsDestroy, track)
	e.GET("/taka2", MessagesIndex, track)
	e.GET("/taka2/", MessagesIndex, track)
	e.GET("/taka2/messages", MessagesIndex, track)
	e.GET("/taka2/messages/", MessagesIndex, track)
	e.GET("/taka2/messages/new", MessagesNew, track)
	e.POST("/taka2/messages", MessagesCreate, track)
	e.GET("/taka2/messages/:id", MessagesShow, track)
	e.GET("/taka2/messages/:id/delete", MessagesDestroy, track)
	e.GET("/taka2/messages/:id/edit", MessagesEdit, track)
	e.POST("/taka2/messages/:id", MessagesUpdate, track)

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		fmt.Println(err)                                    // 標準出力へ
		c.JSON(http.StatusInternalServerError, err.Error()) // ブラウザ画面へ
	}

	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "1323"
	}
	e.Logger.Fatal(e.Start(":" + port))
}

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func Hello(c echo.Context) error {
	return c.Render(http.StatusOK, "hello", "World")
}

func SessionsNew(c echo.Context) error {
	sess, _ := session.Get("session", c)
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
	}
	var s int
	if sess.Values["user_id"] == nil {
		s = 0
	} else {
		s = sess.Values["user_id"].(int)
	}
	return c.Render(http.StatusOK, "sessions_new", struct {
		Session_user_id int
	}{s})
}

func createSessions(c echo.Context) error {
	sess, _ := session.Get("session", c)
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
	}

	password := c.FormValue("password")
	if password == "haruhi" || password == "haluhi" {
		sess.Values["user_id"] = 1
		sess.Save(c.Request(), c.Response())
	} else if password == "uuu" {
		sess.Values["user_id"] = 2
		sess.Save(c.Request(), c.Response())
	} else if password == "nagasaki" || password == "nagahashi" {
		sess.Values["user_id"] = 3
		sess.Save(c.Request(), c.Response())
	} else if password == "uuunagasaki" {
		sess.Values["user_id"] = 4
		sess.Save(c.Request(), c.Response())
	} else {
		return c.Render(http.StatusOK, "sessions_new", struct {
			Session_user_id int
		}{0})
	}

	return c.Redirect(302, "/taka2/messages/")
}

func MessagesIndex(c echo.Context) error {
	const numPerPage = 10
	sess, _ := session.Get("session", c)
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
	}

	var her_id, my_id int
	if sess.Values["user_id"].(int)%2 == 1 {
		her_id = sess.Values["user_id"].(int)
		my_id = sess.Values["user_id"].(int) + 1
	} else {
		her_id = sess.Values["user_id"].(int) - 1
		my_id = sess.Values["user_id"].(int)
	}

	var dbsess db.Session
	var err error
	for i := 0; i < 5; i++ {
		dbsess, err = cockroachdb.Open(settings)
		if err != nil {
			//c.Echo().Logger.Fatal("cockroachdb.Open: ", err)
			log.Print("cockroachdb.Open: ", err)
		} else {
			log.Printf("cockroachdb.Open ok")
			break
		}
	}
	defer dbsess.Close()

	var co int
	rows, err := dbsess.SQL().
		Query(`SELECT COUNT(id) FROM messages WHERE userid = ? OR userid = ?`, her_id, my_id)
	if err != nil {
		log.Fatal("Query: ", err)
	}
	if !rows.Next() {
		log.Fatal("Expecting one row")
	}
	if err := rows.Scan(&co); err != nil {
		log.Fatal("Scan: ", err)
	}
	if err := rows.Close(); err != nil {
		log.Fatal("Close: ", err)
	}
	c.Echo().Logger.Debug("count=" + strconv.Itoa(co))
	var max_page int
	max_page = co / numPerPage
	if co%numPerPage != 0 {
		max_page += 1
	}
	pages := make([]int, max_page)
	for i := 0; i < max_page; i++ {
		pages[i] = i + 1
	}

	var messages []Message
	var page int
	if c.QueryParam("page") == "" || c.QueryParam("page") == "1" {
		page = 1
		rows, err = dbsess.SQL().Query(`SELECT * FROM messages WHERE userid = ? OR userid = ? order by id desc limit ?`, her_id, my_id, numPerPage)
		iter := dbsess.SQL().NewIterator(rows)
		err = iter.All(&messages)
	} else {
		page, _ = strconv.Atoi(c.QueryParam("page"))
		//↓ mysql?
		//db.Raw("SELECT * FROM messages join (select min(id) as cutoff from (select id from messages WHERE userid = ? OR userid = ? order by id desc limit ?) trim) minid on messages.id < minid.cutoff having userid = ? OR userid = ? ORDER BY id desc limit ?", her_id, my_id, (page-1)*numPerPage, her_id, my_id, numPerPage).Scan(&m)
		//↓ postgresql?
		rows, err = dbsess.SQL().Query(`SELECT * FROM messages join (select min(id) as cutoff from (select id from messages WHERE userid = ? OR userid = ? order by id desc limit ?) trim) minid on messages.id < minid.cutoff where userid = ? OR userid = ? ORDER BY id desc limit ?`, her_id, my_id, (page-1)*numPerPage, her_id, my_id, numPerPage)
		iter := dbsess.SQL().NewIterator(rows)
		err = iter.All(&messages)
	}

	return c.Render(http.StatusOK, "messages_index", struct {
		Session_user_id int
		Current_page    int
		Pages           []int
		Mmm             []Message
	}{sess.Values["user_id"].(int), page, pages, messages})
}

func MessagesNew(c echo.Context) error {
	sess, _ := session.Get("session", c)
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
	}
	return c.Render(http.StatusOK, "messages_new", struct {
		Session_user_id int
	}{sess.Values["user_id"].(int)})
}

func MessagesCreate(c echo.Context) error {
	var dbsess db.Session
	var err error
	for i := 0; i < 5; i++ {
		dbsess, err = cockroachdb.Open(settings)
		if err != nil {
			//c.Echo().Logger.Fatal("cockroachdb.Open: ", err)
			log.Print("cockroachdb.Open: ", err)
		} else {
			log.Printf("cockroachdb.Open ok")
			break
		}
	}
	defer dbsess.Close()

	messageCollection := dbsess.Collection("messages")

	sess, _ := session.Get("session", c)
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
	}

	res, err := messageCollection.Insert(Message{
		Userid:    sess.Values["user_id"].(int),
		Body:      c.FormValue("message[body]"),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	if err != nil {
		c.Echo().Logger.Fatal(err)
	}
	return c.Redirect(302, "/taka2/messages/"+fmt.Sprintf("%v", res.ID()))
}

func MessagesShow(c echo.Context) error {
	var dbsess db.Session
	var err error
	for i := 0; i < 5; i++ {
		dbsess, err = cockroachdb.Open(settings)
		if err != nil {
			//c.Echo().Logger.Fatal("cockroachdb.Open: ", err)
			log.Print("cockroachdb.Open: ", err)
		} else {
			log.Printf("cockroachdb.Open ok")
			break
		}
	}
	defer dbsess.Close()

	messageCollection := dbsess.Collection("messages")

	sess, _ := session.Get("session", c)
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
	}

	if regexp.MustCompile(`^[0-9]+$`).MatchString(c.Param("id")) {
		var message Message
		res := messageCollection.Find(db.Cond{"id": c.Param("id")})
		err = res.One(&message)
		return c.Render(http.StatusOK, "messages_show", struct {
			Session_user_id int
			Mmm             Message
		}{sess.Values["user_id"].(int), message})
	} else {
		return c.Render(http.StatusOK, "hello", "World")
	}
}

func MessagesDestroy(c echo.Context) error {
	var dbsess db.Session
	var err error
	for i := 0; i < 5; i++ {
		dbsess, err = cockroachdb.Open(settings)
		if err != nil {
			//c.Echo().Logger.Fatal("cockroachdb.Open: ", err)
			log.Print("cockroachdb.Open: ", err)
		} else {
			log.Printf("cockroachdb.Open ok")
			break
		}
	}
	defer dbsess.Close()

	messageCollection := dbsess.Collection("messages")

	res := messageCollection.Find(db.Cond{"id": c.Param("id")})
	err = res.Delete()

	return c.Redirect(302, "/taka2/messages/")
}

func MessagesEdit(c echo.Context) error {
	var dbsess db.Session
	var err error
	for i := 0; i < 5; i++ {
		dbsess, err = cockroachdb.Open(settings)
		if err != nil {
			//c.Echo().Logger.Fatal("cockroachdb.Open: ", err)
			log.Print("cockroachdb.Open: ", err)
		} else {
			log.Printf("cockroachdb.Open ok")
			break
		}
	}
	defer dbsess.Close()

	messageCollection := dbsess.Collection("messages")

	sess, _ := session.Get("session", c)
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
	}

	var message Message
	res := messageCollection.Find(db.Cond{"id": c.Param("id")})
	err = res.One(&message)
	return c.Render(http.StatusOK, "messages_edit", struct {
		Session_user_id int
		Mmm             Message
	}{sess.Values["user_id"].(int), message})
}

func MessagesUpdate(c echo.Context) error {
	var dbsess db.Session
	var err error
	for i := 0; i < 5; i++ {
		dbsess, err = cockroachdb.Open(settings)
		if err != nil {
			//c.Echo().Logger.Fatal("cockroachdb.Open: ", err)
			log.Print("cockroachdb.Open: ", err)
		} else {
			log.Printf("cockroachdb.Open ok")
			break
		}
	}
	defer dbsess.Close()

	messageCollection := dbsess.Collection("messages")

	var message Message
	res := messageCollection.Find(db.Cond{"id": c.Param("id")})
	err = res.One(&message)
	message.Body = c.FormValue("message[body]")
	message.UpdatedAt = time.Now()
	err = res.Update(message)

	if err != nil {
		c.Echo().Logger.Fatal(err)
	} else {
	}
	return c.Redirect(302, "/taka2/messages/"+c.Param("id"))
}

func SessionsDestroy(c echo.Context) error {
	sess, _ := session.Get("session", c)
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
	}

	if id, _ := strconv.Atoi(c.Param("id")); sess.Values["user_id"].(int) == id {
		println("session clear")
		//session.Set("user_id", 32)
		sess.Values["user_id"] = nil
		sess.Save(c.Request(), c.Response())
	}
	return c.Redirect(302, "/taka2/messages/")
}
