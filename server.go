package main

import (
	//"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gocraft/dbr"
	"github.com/ipfans/echo-session"
	"github.com/labstack/echo"
)

type Messages struct {
	Id         int          `db:"id"`
	Userid     int          `db:"userid"`
	Body       string       `db:"body"`
	Created_at dbr.NullTime `db:"created_at"`
	Updated_at dbr.NullTime `db:"updated_at"`
}

var (
	conn, _ = dbr.Open("mysql", "uuu:oohana@tcp(www5183ui.sakura.ne.jp:3306)/taka", nil)
	sess    = conn.NewSession(nil)
)

func main() {
	e := echo.New()

	store := session.NewCookieStore([]byte("secret"))
	e.Use(session.Sessions("GSESSION", store))

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
		"dt":  func(a dbr.NullTime) string { return a.Time.Format("2006年01月02日, 15:04:05") },
	}

	t := &Template{
		templates: template.Must(template.New("calculator").Funcs(funcMap).ParseGlob("public/views/*.html")),
	}
	e.Renderer = t

	// Route level middleware
	track := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			println("request to /users")
			session := session.Default(c)
			//session.Set("user_id", 1)
			//session.Save()
			var s int
			if session.Get("user_id") == nil {
				s = 0
			} else {
				s = session.Get("user_id").(int)
			}
			if s == 0 {
				return c.Redirect(302, "/taka2/sessions/new")
			}
			return next(c)
		}
	}

	e.GET("/hello", Hello)
	e.GET("/taka2/sessions/new", SessionsNew)
	e.POST("/taka2/sessions", createSessions)
	e.GET("/taka2/sessions/:id/delete", SessionsDestroy)
	e.GET("/taka2/messages", MessagesIndex, track)
	e.GET("/taka2/messages/", MessagesIndex, track)
	e.GET("/taka2/messages/new", MessagesNew)
	e.POST("/taka2/messages", MessagesCreate)
	e.GET("/taka2/messages/:id", MessagesShow)
	e.GET("/taka2/messages/:id/delete", MessagesDestroy)
	e.GET("/taka2/messages/:id/edit", MessagesEdit)
	e.POST("/taka2/messages/:id", MessagesUpdate)

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
	session := session.Default(c)
	var s int
	if session.Get("user_id") == nil {
		s = 0
	} else {
		s = session.Get("user_id").(int)
	}
	return c.Render(http.StatusOK, "sessions_new", struct {
		Session_user_id int
	}{s})
}

func createSessions(c echo.Context) error {
	session := session.Default(c)

	password := c.FormValue("password")
	if password == "sss" {
		session.Set("user_id", 1)
		session.Save()
	} else if password == "uuu" {
		session.Set("user_id", 2)
		session.Save()
	} else {
		return c.Render(http.StatusOK, "sessions_new", struct {
			Session_user_id int
		}{0})
	}

	return c.Redirect(302, "/taka2/messages/")
}

func MessagesIndex(c echo.Context) error {
	println("index")
	var m []Messages
	sess.Select("*").From("messages").OrderBy("id desc").Load(&m)
	session := session.Default(c)

	return c.Render(http.StatusOK, "messages_index", struct {
		Session_user_id int
		Mmm             []Messages
	}{session.Get("user_id").(int), m})
}

func MessagesNew(c echo.Context) error {
	session := session.Default(c)
	return c.Render(http.StatusOK, "messages_new", struct {
		Session_user_id int
	}{session.Get("user_id").(int)})
}

func MessagesCreate(c echo.Context) error {
	session := session.Default(c)
	result, err := sess.InsertInto("messages").
		Columns("userid", "body", "created_at", "updated_at").
		Values(session.Get("user_id").(int), c.FormValue("message[body]"), time.Now(), time.Now()).
		Exec()

	//var count int64
	var lii int64
	if err != nil {
		//log.Fatal(err)
		//count = 1
	} else {
		//count, _ = result.RowsAffected()
		lii, _ = result.LastInsertId()
		//fmt.Println(count) // => 1
	}
	return c.Redirect(302, "/taka2/messages/"+strconv.FormatInt(lii, 10))
}

func MessagesShow(c echo.Context) error {
	session := session.Default(c)
	var m []Messages
	sess.Select("*").From("messages").Where("id = ?", c.Param("id")).Load(&m)
	var mm Messages
	mm = m[0]
	return c.Render(http.StatusOK, "messages_show", struct {
		Session_user_id int
		Mmm             Messages
	}{session.Get("user_id").(int), mm})
}

func MessagesDestroy(c echo.Context) error {
	sess.DeleteFrom("messages").
		Where("id = ?", c.Param("id")).
		Exec()
	return c.Redirect(302, "/taka2/messages/")
}

func MessagesEdit(c echo.Context) error {
	session := session.Default(c)
	var m []Messages
	sess.Select("*").From("messages").Where("id = ?", c.Param("id")).Load(&m)
	var mm Messages
	mm = m[0]
	return c.Render(http.StatusOK, "messages_edit", struct {
		Session_user_id int
		Mmm             Messages
	}{session.Get("user_id").(int), mm})
}

func MessagesUpdate(c echo.Context) error {
	result, err := sess.Update("messages").
		Set("body", c.FormValue("message[body]")).
		Set("updated_at", time.Now()).
		Where("id = ?", c.Param("id")).
		Exec()

	var count int64 = 1
	if err != nil {
		//log.Fatal(err)
	} else {
		count, _ = result.RowsAffected()
		//fmt.Println(count) // => 1
	}
	count = count + 1
	return c.Redirect(302, "/taka2/messages/"+c.Param("id"))
}

func SessionsDestroy(c echo.Context) error {
	session := session.Default(c)

	if id, _ := strconv.Atoi(c.Param("id")); session.Get("user_id").(int) == id {
		println("session clear")
		//session.Set("user_id", 32)
		session.Clear()
		session.Save()
	}
	return c.Redirect(302, "/taka2/messages/")
}
