package main

import (
	"html/template"
	"io"
	"net/http"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gocraft/dbr"
	"github.com/ipfans/echo-session"
	"github.com/labstack/echo"
)

type Messages struct {
	Userid     int          `db:"userid"`
	Body       string       `db:"body"`
	Created_at dbr.NullTime `db:"created_at"`
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
	}

	t := &Template{
		templates: template.Must(template.New("calculator").Funcs(funcMap).ParseGlob("public/views/*.html")),
	}
	e.Renderer = t
	e.GET("/hello", Hello)
	e.GET("/taka2/sessions/new", SessionsNew)
	e.POST("/taka2/sessions", createSessions)
	e.GET("/taka2/messages", MessagesIndex)

	e.Logger.Fatal(e.Start(":1323"))
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
	return c.Render(http.StatusOK, "sessions_new", "World")
}

func createSessions(c echo.Context) error {
	session := session.Default(c)

	password := c.FormValue("password")
	if password == "uuu" {
		session.Set("user_id", 1)
		session.Save()
	} else if password == "sss" {
		session.Set("user_id", 2)
		session.Save()
	} else {
		return c.Render(http.StatusOK, "sessions_new", "World")
	}

	return c.String(http.StatusOK, "password:"+strconv.Itoa(session.Get("user_id").(int)))
}

func MessagesIndex(c echo.Context) error {
	var m []Messages
	sess.Select("*").From("messages").Load(&m)

	session := session.Default(c)

	return c.Render(http.StatusOK, "messages_index", struct {
		Session_user_id int
		Mmm             []Messages
	}{session.Get("user_id").(int), m})
}
