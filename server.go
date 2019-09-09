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
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gocraft/dbr"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	  "github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/labstack/echo/v4"
)

type Messages struct {
	Id         int          `db:"id"`
	Userid     int          `db:"userid"`
	Body       string       `db:"body"`
	Created_at dbr.NullTime `db:"created_at"`
	Updated_at dbr.NullTime `db:"updated_at"`
}

// CREATE TABLE missages (
// 	 id INT NOT NULL,
// 	 userid INT NULL,
// 	 body STRING NULL,
//	 created_at TIMESTAMP NULL,
//	 updated_at TIMESTAMP NULL,
//	 CONSTRAINT "primary" PRIMARY KEY (id ASC),
//	 FAMILY "primary" (id, userid, body, created_at, updated_at)
//	 );
// CREATE SEQUENCE missages_seq;
// show create missages_seq;
// ALTER TABLE missages ALTER COLUMN id SET DEFAULT nextval('missages_seq');
//
// ↓別のAuto Increment方法（これだと、idが484778898812534786のようになる）
//
// CREATE TABLE missages (
// 	 id SERIAL NOT NULL,
// 	 userid INT NULL,
// 	 body STRING NULL,
//	 created_at TIMESTAMP NULL,
//	 updated_at TIMESTAMP NULL,
//	 CONSTRAINT "primary" PRIMARY KEY (id ASC),
//	 FAMILY "primary" (id, userid, body, created_at, updated_at)
//	 );
//
// CREATE USER uuu WITH PASSWORD 'oohana';
// select * from pg_user;
// GRANT ALL ON DATABASE taka TO uuu;
// SHOW GRANTS ON DATABASE taka;
// GRANT ALL ON TABLE taka.* TO uuu;
// SHOW GRANTS ON TABLE taka.*;
type Missage struct {
	Id        int `gorm:"primary_key"` //`gorm:"primary_key;DEFAULT:nextval('messages_seq')"`
	Userid    int
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

var (
	addr string
)

func main() {
	e := echo.New()

	name, err := os.Hostname()
	fmt.Printf("Hostname: %s\n", name)
	if name == "upc" {
		addr = "postgresql://uuu:oohana@35.185.194.40:26257/taka"
	} else {
		addr = "postgresql://uuu:oohana@cockroachdb-public.default.svc.cluster.local:26257/taka"
	}

	db, err := gorm.Open("postgres", addr)
	if err != nil {
		e.Logger.Fatal(err)
	}
	defer db.Close()
	db.AutoMigrate(&Missage{})

	e.Static("/taka2/assets", "assets")

	e.Use(session.Middleware(sessions.NewCookieStore([]byte("secret"))))

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})

	//	e.POST("/users", saveUser)
	e.GET("/users/:id", getUser)
	e.GET("/show", show)
	e.POST("/save", save)
	//	e.PUT("/users/:id", updateUser)
	//	e.DELETE("/users/:id", deleteUser)

	e.Static("/static", "static")

	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"mul": func(a, b int) int { return a * b },
		"div": func(a, b int) int { return a / b },
		"mod": func(a, b int) int { return a % b },
		"br":  func(a string) string { return strings.Replace(a, "\n", "<br/>", -1) },
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
	e.GET("/taka2/suusiki", Suusiki)

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

// e.GET("/users/:id", getUser)
func getUser(c echo.Context) error {
	// User ID from path `users/:id`
	id := c.Param("id")
	return c.String(http.StatusOK, id)
}

//e.GET("/show", show)
func show(c echo.Context) error {
	// Get team and member from the query string
	team := c.QueryParam("team")
	member := c.QueryParam("member")
	return c.String(http.StatusOK, "team:"+team+", member:"+member)
}

func save(c echo.Context) error {
	// Get name and email
	name := c.FormValue("name")
	email := c.FormValue("email")
	return c.String(http.StatusOK, "name:"+name+", email:"+email)
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
	if password == "sena" || password == "Sena" {
		sess.Values["user_id"] = 1
		sess.Save(c.Request(), c.Response())
	} else if password == "uuu" {
		sess.Values["user_id"] = 2
		sess.Save(c.Request(), c.Response())
	} else if password == "aaa" {
		sess.Values["user_id"] = 3
		sess.Save(c.Request(), c.Response())
	} else if password == "uuuaaa" {
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

	db, err := gorm.Open("postgres", addr)
	if err != nil {
		c.Echo().Logger.Fatal(err)
	}
	defer db.Close()

	var co int
	//sess.Select("count(id)").From("messages").Where("userid = ? OR userid = ?", her_id, my_id).Load(&co)
	db.Model(&Missage{}).Where("userid = ?", her_id).Or("userid = ?", my_id).Count(&co)
	c.Echo().Logger.Debug("count=" + strconv.Itoa(co))
	//// SELECT count(*) FROM users WHERE name = 'jinzhu'; (count)
	var max_page int
	max_page = co / numPerPage
	if co%numPerPage != 0 {
		max_page += 1
	}
	pages := make([]int, max_page)
	for i := 0; i < max_page; i++ {
		pages[i] = i + 1
	}

	var m []Missage
	var page int
	if c.QueryParam("page") == "" || c.QueryParam("page") == "1" {
		page = 1
		//sess.SelectBySql("SELECT * FROM messages WHERE userid = ? OR userid = ? ORDER BY id desc limit ?", her_id, my_id, numPerPage).Load(&m)
		db.Order("id desc").Limit(numPerPage).Where("userid = ? OR userid = ?", her_id, my_id).Find(&m)
	} else {
		page, _ = strconv.Atoi(c.QueryParam("page"))
		//sess.SelectBySql("SELECT * FROM messages join (select min(id) as cutoff from (select id from messages WHERE userid = ? OR userid = ? order by id desc limit ?) trim) minid on messages.id < minid.cutoff having userid = ? OR userid = ? ORDER BY id desc limit ?", her_id, my_id, (page-1)*numPerPage, her_id, my_id, numPerPage).Load(&m)

		//↓ mysql?
		//db.Raw("SELECT * FROM missages join (select min(id) as cutoff from (select id from missages WHERE userid = ? OR userid = ? order by id desc limit ?) trim) minid on missages.id < minid.cutoff having userid = ? OR userid = ? ORDER BY id desc limit ?", her_id, my_id, (page-1)*numPerPage, her_id, my_id, numPerPage).Scan(&m)
		//↓ postgresql?
		db.Raw("SELECT * FROM missages join (select min(id) as cutoff from (select id from missages WHERE userid = ? OR userid = ? order by id desc limit ?) trim) minid on missages.id < minid.cutoff where userid = ? OR userid = ? ORDER BY id desc limit ?", her_id, my_id, (page-1)*numPerPage, her_id, my_id, numPerPage).Scan(&m)
	}
	//sess.Select("*").From("messages").Where("userid = ? OR userid = ?", her_id, my_id).OrderBy("id desc").Load(&m)

	return c.Render(http.StatusOK, "messages_index", struct {
		Session_user_id int
		Current_page    int
		Pages           []int
		Mmm             []Missage
	}{sess.Values["user_id"].(int), page, pages, m})
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
	db, err := gorm.Open("postgres", addr)
	if err != nil {
		c.Echo().Logger.Fatal(err)
	}
	defer db.Close()

	sess, _ := session.Get("session", c)
	sess.Options = &sessions.Options{
	  Path:     "/",
	  MaxAge:   86400 * 7,
	  HttpOnly: true,
	}
	//result, err := sess.InsertInto("messages").
	//	Columns("userid", "body", "created_at", "updated_at").
	//	Values(sess.Values["user_id"].(int), c.FormValue("message[body]"), time.Now(), time.Now()).
	//	Exec()
	m := Missage{Userid: sess.Values["user_id"].(int), Body: c.FormValue("message[body]"), CreatedAt: time.Now(), UpdatedAt: time.Now()}
	db.Create(&m)

	//var count int64
	var lii int64
	if err != nil {
		//log.Fatal(err)
		//count = 1
	} else {
		//count, _ = result.RowsAffected()
		lii = int64(m.Id)
		//fmt.Println(count) // => 1
	}
	return c.Redirect(302, "/taka2/messages/"+strconv.FormatInt(lii, 10))
}

func MessagesShow(c echo.Context) error {
	db, err := gorm.Open("postgres", addr)
	if err != nil {
		c.Echo().Logger.Fatal(err)
	}
	defer db.Close()

	sess, _ := session.Get("session", c)
	sess.Options = &sessions.Options{
	  Path:     "/",
	  MaxAge:   86400 * 7,
	  HttpOnly: true,
	}
	//sess.Select("*").From("messages").Where("id = ?", c.Param("id")).Load(&m)
	var mm Missage
	db.Where("id = ?", c.Param("id")).First(&mm)
	return c.Render(http.StatusOK, "messages_show", struct {
		Session_user_id int
		Mmm             Missage
	}{sess.Values["user_id"].(int), mm})
}

func Suusiki(c echo.Context) error {
	db, err := gorm.Open("postgres", addr)
	if err != nil {
		c.Echo().Logger.Fatal(err)
	}
	defer db.Close()

	m := Missage{Userid: 2, Body: c.QueryParam("page"), CreatedAt: time.Now(), UpdatedAt: time.Now()}
	db.Create(&m)

	//sess.Select("*").From("messages").Where("id = ?", c.Param("id")).Load(&m)
	var mm Missage
	//db.Where("id = ?", c.Param("id")).First(&mm)
	return c.Render(http.StatusOK, "suusiki", struct {
		Session_user_id int
		Mmm             Missage
	}{1, mm})
}

func MessagesDestroy(c echo.Context) error {
	db, err := gorm.Open("postgres", addr)
	if err != nil {
		c.Echo().Logger.Fatal(err)
	}
	defer db.Close()

	//sess.DeleteFrom("messages").
	//	Where("id = ?", c.Param("id")).
	//	Exec()
	db.Where("id = ?", c.Param("id")).Delete(&Missage{})

	return c.Redirect(302, "/taka2/messages/")
}

func MessagesEdit(c echo.Context) error {
	db, err := gorm.Open("postgres", addr)
	if err != nil {
		c.Echo().Logger.Fatal(err)
	}
	defer db.Close()

	sess, _ := session.Get("session", c)
	sess.Options = &sessions.Options{
	  Path:     "/",
	  MaxAge:   86400 * 7,
	  HttpOnly: true,
	}
	//var m []Messages
	//sess.Select("*").From("messages").Where("id = ?", c.Param("id")).Load(&m)
	var mm Missage
	db.Where("id = ?", c.Param("id")).First(&mm)
	return c.Render(http.StatusOK, "messages_edit", struct {
		Session_user_id int
		Mmm             Missage
	}{sess.Values["user_id"].(int), mm})
}

func MessagesUpdate(c echo.Context) error {
	db, err := gorm.Open("postgres", addr)
	if err != nil {
		c.Echo().Logger.Fatal(err)
	}
	defer db.Close()

	//result, err := sess.Update("messages").
	//	Set("body", c.FormValue("message[body]")).
	//	Set("updated_at", time.Now()).
	//	Where("id = ?", c.Param("id")).
	//	Exec()

	var mm Missage
	db.Where("id = ?", c.Param("id")).First(&mm)
	mm.Body = c.FormValue("message[body]")
	mm.UpdatedAt = time.Now()
	db.Save(&mm)

	var count int64 = 1
	if err != nil {
		//log.Fatal(err)
	} else {
		//count, _ = result.RowsAffected()
		//fmt.Println(count) // => 1
	}
	count = count + 1
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
