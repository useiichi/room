package main

import (
	//"fmt"
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
	"github.com/ipfans/echo-session"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/labstack/echo"
)

type Messages struct {
	Id         int          `db:"id"`
	Userid     int          `db:"userid"`
	Body       string       `db:"body"`
	Created_at dbr.NullTime `db:"created_at"`
	Updated_at dbr.NullTime `db:"updated_at"`
}

// ALTER TABLE missages ALTER COLUMN id SET DEFAULT nextval('messages_seq');
type Missage struct {
	Id        int `gorm:"primary_key"` //`gorm:"primary_key;DEFAULT:nextval('messages_seq')"`
	Userid    int
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

var (
	conn, _ = dbr.Open("mysql", "uuu:oohana@tcp(proxysql-svc.default.svc.cluster.local:6033)/taka", nil)
	sess    = conn.NewSession(nil)
)

func main() {
	e := echo.New()

	//const addr = "postgresql://uuu:oohana@mail.iseisaku.com:26257/taka"
	const addr = "postgresql://uuu:oohana@cockroachdb-public.default.svc.cluster.local:26257/taka"
	db, err := gorm.Open("postgres", addr)
	if err != nil {
		e.Logger.Fatal(err)
	}
	defer db.Close()
	db.AutoMigrate(&Missage{})

	e.Static("/taka2/assets", "assets")

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
			session := session.Default(c)
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
	if password == "lamu" || password == "ramu" {
		session.Set("user_id", 1)
		session.Save()
	} else if password == "uuu" {
		session.Set("user_id", 2)
		session.Save()
	} else if password == "aaa" {
		session.Set("user_id", 3)
		session.Save()
	} else if password == "uuuaaa" {
		session.Set("user_id", 4)
		session.Save()
	} else {
		return c.Render(http.StatusOK, "sessions_new", struct {
			Session_user_id int
		}{0})
	}

	return c.Redirect(302, "/taka2/messages/")
}

func MessagesIndex(c echo.Context) error {
	const numPerPage = 10
	session := session.Default(c)
	var her_id, my_id int
	if session.Get("user_id").(int)%2 == 1 {
		her_id = session.Get("user_id").(int)
		my_id = session.Get("user_id").(int) + 1
	} else {
		her_id = session.Get("user_id").(int) - 1
		my_id = session.Get("user_id").(int)
	}

	//const addr = "postgresql://uuu:oohana@mail.iseisaku.com:26257/taka"
	const addr = "postgresql://uuu:oohana@cockroachdb-public.default.svc.cluster.local:26257/taka"
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
	}{session.Get("user_id").(int), page, pages, m})
}

func MessagesNew(c echo.Context) error {
	session := session.Default(c)
	return c.Render(http.StatusOK, "messages_new", struct {
		Session_user_id int
	}{session.Get("user_id").(int)})
}

func MessagesCreate(c echo.Context) error {
	//const addr = "postgresql://uuu:oohana@mail.iseisaku.com:26257/taka"
	const addr = "postgresql://uuu:oohana@cockroachdb-public.default.svc.cluster.local:26257/taka"
	db, err := gorm.Open("postgres", addr)
	if err != nil {
		c.Echo().Logger.Fatal(err)
	}
	defer db.Close()

	session := session.Default(c)
	//result, err := sess.InsertInto("messages").
	//	Columns("userid", "body", "created_at", "updated_at").
	//	Values(session.Get("user_id").(int), c.FormValue("message[body]"), time.Now(), time.Now()).
	//	Exec()
	m := Missage{Userid: session.Get("user_id").(int), Body: c.FormValue("message[body]"), CreatedAt: time.Now(), UpdatedAt: time.Now()}
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
	//const addr = "postgresql://uuu:oohana@mail.iseisaku.com:26257/taka"
	const addr = "postgresql://uuu:oohana@cockroachdb-public.default.svc.cluster.local:26257/taka"
	db, err := gorm.Open("postgres", addr)
	if err != nil {
		c.Echo().Logger.Fatal(err)
	}
	defer db.Close()

	session := session.Default(c)
	//sess.Select("*").From("messages").Where("id = ?", c.Param("id")).Load(&m)
	var mm Missage
	db.Where("id = ?", c.Param("id")).First(&mm)
	return c.Render(http.StatusOK, "messages_show", struct {
		Session_user_id int
		Mmm             Missage
	}{session.Get("user_id").(int), mm})
}

func MessagesDestroy(c echo.Context) error {
	//const addr = "postgresql://uuu:oohana@mail.iseisaku.com:26257/taka"
	const addr = "postgresql://uuu:oohana@cockroachdb-public.default.svc.cluster.local:26257/taka"
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
	//const addr = "postgresql://uuu:oohana@mail.iseisaku.com:26257/taka"
	const addr = "postgresql://uuu:oohana@cockroachdb-public.default.svc.cluster.local:26257/taka"
	db, err := gorm.Open("postgres", addr)
	if err != nil {
		c.Echo().Logger.Fatal(err)
	}
	defer db.Close()

	session := session.Default(c)
	//var m []Messages
	//sess.Select("*").From("messages").Where("id = ?", c.Param("id")).Load(&m)
	var mm Missage
	db.Where("id = ?", c.Param("id")).First(&mm)
	return c.Render(http.StatusOK, "messages_edit", struct {
		Session_user_id int
		Mmm             Missage
	}{session.Get("user_id").(int), mm})
}

func MessagesUpdate(c echo.Context) error {
	//const addr = "postgresql://uuu:oohana@mail.iseisaku.com:26257/taka"
	const addr = "postgresql://uuu:oohana@cockroachdb-public.default.svc.cluster.local:26257/taka"
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
	session := session.Default(c)

	if id, _ := strconv.Atoi(c.Param("id")); session.Get("user_id").(int) == id {
		println("session clear")
		//session.Set("user_id", 32)
		session.Clear()
		session.Save()
	}
	return c.Redirect(302, "/taka2/messages/")
}
