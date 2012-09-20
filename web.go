package main

import (
	"code.google.com/p/gorilla/sessions"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var store = sessions.NewCookieStore([]byte("something-very-secret-for-session"))

func webmain() {
	http.Handle("/js/", http.FileServer(http.Dir("templates")))
	http.Handle("/css/", http.FileServer(http.Dir("templates")))
	http.Handle("/pic/", http.FileServer(http.Dir("templates")))

	http.HandleFunc("/login/", loginHandler)
	http.HandleFunc("/register/", registerHandler)
	http.HandleFunc("/user/", userHandler)
	http.HandleFunc("/logout/", logoutHandler)
	http.HandleFunc("/reset/", resetHandler)
	http.HandleFunc("/add/", addHandler)
	http.HandleFunc("/todo/", todoHandler)
	http.HandleFunc("/history/", historyHandler)
	http.HandleFunc("/bye/", byeHandler)
	http.HandleFunc("/", NotFoundHandler)

	ln, err := net.Listen("tcp", host+":"+webport)
	if err != nil {
		LogW("listen error on port", webport)
		closeFdChan <- true
		mainChan <- true
		return
	}
	defer ln.Close()

	closeFdChan <- true
	http.Serve(ln, nil)
}

func loginHandler(rw http.ResponseWriter, req *http.Request) {
	LogW(req.Host, req.Method, req.RequestURI, req.RemoteAddr, req.UserAgent(), req.Referer())
	type AuthError struct {
		ErrStr string
	}

	if CheckSessionForUser(req) {
		http.Redirect(rw, req, "/user/", http.StatusFound)
		return
	}

	if req.Method == "POST" {
		username := req.FormValue("username")
		password := req.FormValue("password")
		if username != "" && password != "" {
			if uid, err := redisClient.Get("user:" + username); err == nil {
				passwordInDB, _ := redisClient.Get("user:" + string(uid) + ":pass")
				if string(passwordInDB) == password {
					session, _ := store.Get(req, "session")
					session.Values["username"] = username
					session.Values["id"] = string(uid)
					session.Values["logged"] = "1"
					session.Save(req, rw)
					http.Redirect(rw, req, "/user/", http.StatusFound)
					return
				}
			}

			t, _ := template.ParseFiles("templates/html/login.html")
			t.Execute(rw, &AuthError{"用户名或密码错误"})
			return
		}
	}

	t, _ := template.ParseFiles("templates/html/login.html")
	t.Execute(rw, &AuthError{""})
}

func registerHandler(rw http.ResponseWriter, req *http.Request) {
	LogW(req.Host, req.Method, req.RequestURI, req.RemoteAddr, req.UserAgent(), req.Referer())
	if req.Method == "POST" {
		username := req.FormValue("username")
		password := req.FormValue("password")
		passwordrepeat := req.FormValue("passwordrepeat")
		email := req.FormValue("email")

		if password == passwordrepeat && username+passwordrepeat+password+email != "" {
			if ok, _ := redisClient.Exists("user:" + username); ok {
				fmt.Fprintf(rw, "username already exists")
				return
			}

			uid, err := redisClient.Incr("user:next:uid")
			if err != nil {
				fmt.Fprintf(rw, "error")
				return
			}
			usernameKey := "user:" + strconv.FormatInt(uid, 10) + ":name"
			passwordKey := "user:" + strconv.FormatInt(uid, 10) + ":pass"
			emailKey := "user:" + strconv.FormatInt(uid, 10) + ":email"
			uidkey := "user:" + username
			redisClient.Set(usernameKey, []byte(username))
			redisClient.Set(passwordKey, []byte(password))
			redisClient.Set(emailKey, []byte(email))
			redisClient.Set(uidkey, []byte(strconv.FormatInt(uid, 10)))
			http.Redirect(rw, req, "/login/", http.StatusFound)
		}
	}

	t, _ := template.ParseFiles("templates/html/register.html")
	t.Execute(rw, nil)
}

type TodoNums struct {
	TodoNums string
}

func userHandler(rw http.ResponseWriter, req *http.Request) {
	LogW(req.Host, req.Method, req.RequestURI, req.RemoteAddr, req.UserAgent(), req.Referer())

	uid, ok := CheckSessionForLogin(req)
	if !ok {
		http.Redirect(rw, req, "/login/", http.StatusFound)
		return
	}

	num, _ := redisClient.Zcard("comm:" + uid + ":todocids")

	t, _ := template.ParseFiles("templates/html/user.html")
	t.Execute(rw, &TodoNums{strconv.Itoa(num)})
}

func logoutHandler(rw http.ResponseWriter, req *http.Request) {
	LogW(req.Host, req.Method, req.RequestURI, req.RemoteAddr, req.UserAgent(), req.Referer())
	session, _ := store.Get(req, "session")
	session.Values["logged"] = "0"
	// session.Options = &sessions.Options{MaxAge: -1}
	session.Save(req, rw)
	http.Redirect(rw, req, "/login/", http.StatusFound)
}

type ResetError struct {
	ErrStr string
}

func resetHandler(rw http.ResponseWriter, req *http.Request) {
	LogW(req.Host, req.Method, req.RequestURI, req.RemoteAddr, req.UserAgent(), req.Referer())
	uid, ok := CheckSessionForLogin(req)
	if !ok {
		http.Redirect(rw, req, "/login/", http.StatusFound)
		return
	}

	data := &ResetError{""}

	if req.Method == "POST" {
		originpassword := req.FormValue("originpassword")
		password := req.FormValue("password")
		passwordrepeat := req.FormValue("passwordrepeat")

		passwordInDB, _ := redisClient.Get("user:" + uid + ":pass")
		if password == passwordrepeat && originpassword == string(passwordInDB) {
			redisClient.Set("user:"+uid+":pass", []byte(password))
			http.Redirect(rw, req, "/user/", http.StatusFound)
			return
		}
		data = &ResetError{"发生错误"}
	}

	t, _ := template.ParseFiles("templates/html/reset.html")
	t.Execute(rw, data)
}

func addHandler(rw http.ResponseWriter, req *http.Request) {
	LogW(req.Host, req.Method, req.RequestURI, req.RemoteAddr, req.UserAgent(), req.Referer())

	uid, ok := CheckSessionForLogin(req)
	if !ok {
		http.Redirect(rw, req, "/login/", http.StatusFound)
		return
	}

	if req.Method == "POST" {
		command := req.FormValue("command")
		icid, err := redisClient.Incr("comm:next:cid")
		if err != nil {
			fmt.Fprintf(rw, "error")
			return
		}
		cid := strconv.FormatInt(icid, 10)
		redisClient.Set("comm:"+cid+":uid", []byte(uid))
		redisClient.Set("comm:"+cid+":comm", []byte(command))
		redisClient.Set("comm:"+cid+":time", []byte(time.Now().Format("2006-01-02 15:04:05")))
		redisClient.Set("comm:"+cid+":done", []byte("0"))
		redisClient.Set("comm:"+cid+":result", []byte(""))
		score, _ := strconv.Atoi(cid)
		redisClient.Zadd("comm:"+uid+":todocids", []byte(cid), float64(score))

		AddEventFromWeb(uid, cid, command)

		http.Redirect(rw, req, "/user/", http.StatusFound)
		return
	}

	t, _ := template.ParseFiles("templates/html/add.html")
	t.Execute(rw, nil)
}

type TodoInfo struct {
	Command string
	Time    string
}

type TemplateTodoData struct {
	TodoMap      map[string]*TodoInfo
	CurrentPage  int
	PageNum      int
	NextPage     int
	PreviousPage int
}

func todoHandler(rw http.ResponseWriter, req *http.Request) {
	LogW(req.Host, req.Method, req.RequestURI, req.RemoteAddr, req.UserAgent(), req.Referer())

	uid, ok := CheckSessionForLogin(req)
	if !ok {
		http.Redirect(rw, req, "/login/", http.StatusFound)
		return
	}

	spage := req.URL.Path[6:]
	if spage == "" {
		spage = "1"
	}
	ipage, err := strconv.Atoi(spage)
	if err != nil {
		fmt.Fprintf(rw, "error")
		return
	}
	countNum, _ := redisClient.Zcard("comm:" + uid + ":todocids")
	pageNum := countNum / 5
	if countNum%5 != 0 {
		pageNum = pageNum + 1
	}

	if ipage > pageNum {
		ipage = 1
	}
	if countNum == 0 {
		pageNum = 1
	}

	nextPage, previousPage := 1, pageNum
	if ipage+1 <= pageNum {
		nextPage = ipage + 1
	}
	if ipage-1 > 0 {
		previousPage = ipage - 1
	}

	cids, _ := redisClient.Zrange("comm:"+uid+":todocids", (ipage-1)*5, ipage*5-1)
	commMap := make(map[string]*TodoInfo)
	for _, v := range cids {
		command, _ := redisClient.Get("comm:" + string(v) + ":comm")
		ctime, _ := redisClient.Get("comm:" + string(v) + ":time")
		todoInfo := &TodoInfo{string(command), string(ctime)}
		commMap[string(v)] = todoInfo
	}

	t, _ := template.ParseFiles("templates/html/todo.html")
	t.Execute(rw, &TemplateTodoData{commMap, ipage, pageNum, nextPage, previousPage})
}

type HistoryInfo struct {
	Command string
	Time    string
	Result  string
}

type TemplateHistoryData struct {
	HistoryMap   map[string]*HistoryInfo
	CurrentPage  int
	PageNum      int
	NextPage     int
	PreviousPage int
}

func historyHandler(rw http.ResponseWriter, req *http.Request) {
	LogW(req.Host, req.Method, req.RequestURI, req.RemoteAddr, req.UserAgent(), req.Referer())

	uid, ok := CheckSessionForLogin(req)
	if !ok {

		http.Redirect(rw, req, "/login/", http.StatusFound)
		return
	}

	spage := req.URL.Path[9:]
	if spage == "" {
		spage = "1"
	}
	ipage, err := strconv.Atoi(spage)
	if err != nil {
		fmt.Fprintf(rw, "error")
		return
	}
	countNum, _ := redisClient.Zcard("comm:" + uid + ":donecids")
	pageNum := countNum / 5
	if countNum%5 != 0 {
		pageNum = pageNum + 1
	}

	if ipage > pageNum {
		ipage = 1
	}
	if countNum == 0 {
		pageNum = 1
	}
	nextPage, previousPage := 1, pageNum
	if ipage+1 <= pageNum {
		nextPage = ipage + 1
	}
	if ipage-1 > 0 {
		previousPage = ipage - 1
	}

	cids, _ := redisClient.Zrange("comm:"+uid+":donecids", (ipage-1)*5, ipage*5-1)
	commMap := make(map[string]*HistoryInfo)
	for _, v := range cids {
		command, _ := redisClient.Get("comm:" + string(v) + ":comm")
		ctime, _ := redisClient.Get("comm:" + string(v) + ":time")
		result, _ := redisClient.Get("comm:" + string(v) + ":result")
		todoInfo := &HistoryInfo{string(command), string(ctime), string(result)}
		commMap[string(v)] = todoInfo
	}

	t, _ := template.ParseFiles("templates/html/history.html")
	t.Execute(rw, &TemplateHistoryData{commMap, ipage, pageNum, nextPage, previousPage})
}

func byeHandler(rw http.ResponseWriter, req *http.Request) {
	LogW(req.Host, req.Method, req.RequestURI, req.RemoteAddr, req.UserAgent(), req.Referer())
	remoteIP := strings.Split(req.RemoteAddr, ":")[0]
	if (host == "" && remoteIP == "127.0.0.1") || (host == remoteIP) {
		mainChan <- true
		mainChan <- true
	}
}

func NotFoundHandler(rw http.ResponseWriter, req *http.Request) {
	LogW(req.Host, req.Method, req.RequestURI, req.RemoteAddr, req.UserAgent(), req.Referer())
	if req.URL.Path == "/" {
		http.Redirect(rw, req, "/login/", http.StatusFound)
		return
	}

	t, _ := template.ParseFiles("templates/html/404.html")
	t.Execute(rw, nil)
}

func CheckSessionForLogin(req *http.Request) (string, bool) {
	session, _ := store.Get(req, "session")
	logged := session.Values["logged"]
	uid := session.Values["id"]
	if logged == nil || logged.(string) != "1" {
		return "", false
	}
	return uid.(string), true
}

func CheckSessionForUser(req *http.Request) bool {
	session, _ := store.Get(req, "session")
	logged := session.Values["logged"]

	if logged != nil && logged.(string) == "1" {
		return true
	}
	return false
}
