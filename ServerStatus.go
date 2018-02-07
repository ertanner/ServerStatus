package main

import (
	"encoding/json"
	"flag"
	"fmt"
	//"github.com/go-ldap/ldap"
	"gopkg.in/ldap.v2"
	"github.com/gorilla/securecookie"
	//"github.com/gorilla/websocket"

	ps "github.com/gorillalabs/go-powershell"
	"github.com/gorillalabs/go-powershell/backend"
	"github.com/gorillalabs/go-powershell/middleware"

	"github.com/julienschmidt/httprouter"

	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"
	//"time"
)

type Service struct {
	ID           string `json: "id"`
	Env          string `json: "env"`
	ComputerName string `json: "ComputerName"`
	Service      string `json: "Service"`
	Description  string `json: "Description"`
	Status       string
}

var Services []Service
var Environment []string
var ServOrder []Service


var cookieHandler = securecookie.New(
	securecookie.GenerateRandomKey(64),
	securecookie.GenerateRandomKey(32))

var count int
var tpl *template.Template
var err error
var r = httprouter.New()
var userName string
var URLName string

type Web struct {
	URLName  string
	Services []Service
	Environment []string
}

//var upgrader = websocket.Upgrader{
//	ReadBufferSize:  1024,
//	WriteBufferSize: 1024,
//}
//var clients = make(map[*websocket.Conn]bool) // connected clients
//var broadcast = make(chan Message)           // broadcast channel

type Message struct {
	id string `json:"id"`
	env string `'json:"env"`
	status string `json:"status"`
}

type msg struct {
	Num int
}

//var AWeb Web
var debug bool
var boolFlag = flag.Bool("d", false, "debug")
var skipLoad = flag.Bool("s", true, "Skip loading on startup")

const indexPage = `
	<h1 align="center">Server Status</h1><br>
	<h1>Login</h1>
	<form method="post" action="/logon">
		 <label for="name">User name</label>
		 <input type="text" id="name" name="name">
		 <label for="password">Password</label>
		 <input type="password" id="password" name="password">
		 <button type="submit">Login</button>
	</form>
	`

func init() {
	tpl = template.Must(template.ParseGlob("html/*.html"))
	debug = false //turned of by default

}

//+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
// Main program.  Sets the path and calls the ListenAndServe function
func main() {
	//+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
	// file logging
	//+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
	f, err := os.OpenFile("ServerStatus.log", os.O_APPEND | os.O_CREATE | os.O_RDWR, 0666)
	if err != nil {
		fmt.Printf("error opening file: %v", err)
	}else{
		fmt.Println("Open file for logs")
	}
	defer f.Close()
	log.SetOutput(f)

	//+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
	// Parse and process the flags
	//+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
	flag.Parse()
	if *boolFlag {
		debug = true
		fmt.Println("Debug = " + strconv.FormatBool(debug))
	}
	if debug {
		log.Println("got to main")
	}
	//+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
	//Get the JSON and ENVINOMENTS
	//+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
	getJson()
	setEnv()
	//+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
	// if -s
	//+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
	if *skipLoad{
		log.Println("Load flag = false")


	}else{
		log.Println("Load flag = true")
		getStatuses()
	}


	r.GET("/", indexPageHandler)
	//r.GET("/ws",handleConnections)
	r.GET("/home", HomePage)
	r.POST("/logon", logonHandler)
	r.GET("/prod", ServicePage)
	r.GET("/rep", ServicePage)
	r.GET("/test", ServicePage)
	r.GET("/dev", ServicePage)
	r.GET("/project", ServicePage)
	r.GET("/refresh", RefreshList)
	r.GET("/refreshPage", RefreshPage)
	r.GET("/refreshStatus", refreshSatus)
	r.POST("/updateStatus", updateStatus)

	r.ServeFiles("/html/*filepath", http.Dir("html"))
	r.ServeFiles("/js/*filepath", http.Dir("js"))
	r.ServeFiles("/css/*filepath", http.Dir("css"))
	fmt.Println("Ready for web")
	log.Println("Ready for web")

	//go handleMessages(Message{})

	go http.ListenAndServe(":8080", r)
	error:= http.ListenAndServeTLS(":443", "server.crt", "server.key", nil)
	if error!=nil{
		fmt.Println(error)
		log.Fatal("LisenAndServer: ", error)
	}
}

//+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
// This will refresh the entire list of servers with updated statuses for the services
//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
func RefreshList(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	log.Println("Got to Refhreshlist")
	Services = Services[:0]
	Environment = Environment[:0]
	ServOrder = ServOrder[:0]
	getJson()
	setEnv()
	getStatuses()
	log.Println("all lists have been refreshed")
	http.Redirect(w, req, "/", 302)
}


//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
// This will refresh the entire list of servers with updated statuses for the services
//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
func RefreshPage(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	log.Println("got to RefreshPage")
	env := req.URL.Query().Get("env")
	getEnvStatuses(env)
	log.Println("All status for ", env, " have been refreshed")
	http.Redirect(w, req, "/"+env, 302)
}
//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
// This will bring up the home page when called by /home
//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
func HomePage(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	if debug {
		log.Println("got to HomePage")
	}

	if userName != "" {
		log.Println("User name is ", userName)
		tpl.ExecuteTemplate(w, "ServerStatus.html", Environment)
		if err != nil {
			log.Fatalln(err)
		}
	} else {
		log.Println("got to else in HomePage")
		http.Redirect(w, req, "/", 302)
	}
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
// This is the meat of the prcess.  It will display the individual environments services
//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
func ServicePage(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	if debug {
		log.Println("got to ServicePage")
	}
	URLName = strings.Trim(req.URL.String(), "/")

	if debug {
		log.Println(URLName)
	}
	AWeb := Web{URLName, Services, Environment}
	if debug {
		log.Println(Web{})
	}
	if userName != "" {
		log.Println("Loading ServerStatus2.html")
		tpl.ExecuteTemplate(w, "ServerStatus2.html", AWeb)
		if err != nil {
			log.Fatalln(err)
		}
	} else {
		http.Redirect(w, req, "/", 302)
	}

}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
// This open and read the JSON file into an array called services which
// contains individual sturct's of type service
//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
func getJson() {
	file, e := ioutil.ReadFile("./servers.json")
	if e != nil {
		fmt.Sprintf("File error: %v\n", e)
		os.Exit(1)
	}
	var serviceMap []map[string]interface{}

	json.Unmarshal(file, &serviceMap)
	if err != nil {
		panic(err)
	}
	count = 0
	for _, serviceData := range serviceMap {
		count = count + 1
		var s Service
		s.ID = fmt.Sprintf("%s", serviceData["id"])
		s.Env = fmt.Sprintf("%s", serviceData["env"])
		s.Service = fmt.Sprintf("%s", serviceData["Service"])
		s.ComputerName = fmt.Sprintf("%s", serviceData["ComputerName"])
		s.Description = fmt.Sprintf("%s", serviceData["Description"])
		Services = append(Services, s)
	}
	if debug {
		log.Println("line count Services = " + strconv.Itoa(len(Services)))
	}
	//ServOrder =  orderServ(Services)
	if debug {
		log.Println("line count ServOrder = " + strconv.Itoa(len(ServOrder)))
	}
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
// This function parses the array Serivces and creates an array called env
//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
func setEnv() {
	if debug {
		log.Println("got to setEnv")
	}
	var env string

	for i := 0; i < count; i++ {
		env = fmt.Sprintf("%s", Services[i].Env)
		if i == 0 {
			Environment = append(Environment, env)
		}
		var in bool
		in = false
		for j := 0; j < len(Environment); j++ {
			if Environment[j] == env {
				in = true
				break
			}
		}
		if in == false {
			Environment = append(Environment, env)
		}
		in = false
	}
	if debug {
		log.Println(Environment)
	}
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
// this gets the status of each service
//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
func getStatuses() {
	if debug {
		log.Println("Got to getStatus")
	}
	for i := 0; i < len(Services); i++ {
		Services[i].Status = ""
		back := &backend.Local{}
		shell, err := ps.New(back)
		if err != nil {
			log.Println(err)
		}
		config := middleware.NewSessionConfig()
		config.ComputerName = Services[i].ComputerName
		if debug {
			log.Println(config)
		}

		defer shell.Exit()

		serviceName := Services[i].Service
		wmi := " Get-Service -Name \"" + serviceName + "\"  -ComputerName " + Services[i].ComputerName
		if debug {
			log.Println(wmi)
		}
		stdout, stderr, err := shell.Execute(wmi)
		if err != nil {
			log.Println(err)
		}
		if stderr != "" {
			log.Println(stderr)
		}
		if strings.Contains(stdout, "Running") {
			Services[i].Status = "up"
			log.Println("status up")
		} else if strings.Contains(stdout, "down") {
			Services[i].Status = "dn"
			log.Println("status down")
		} else {
			log.Println("status unk")
			Services[i].Status = "ukn"
		}
		log.Println(Services[i].Status)
	}
	log.Println("Services have been registered.")
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
// this will refresh the all of the statuses
//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
func updateStatus(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	log.Println("got to updateStatus")
	//fmt.Println("got to updateStatus")
	// Once done we need to redirect back to the same page so

	req.ParseForm()
	productsSelected := req.Form["check"]

	log.Println(productsSelected)
	log.Println("Request URI " + URLName)
	log.Println("prod sel size " + strconv.Itoa(len(productsSelected)))

	for i := 0; i < len(productsSelected); i++ {
		log.Println("prod selected " + productsSelected[i])
		startStopService(productsSelected[i])
	}
	redirectTarget := URLName
	http.Redirect(w, req, redirectTarget, 302)
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
// this will refresh the status for a specific enviornment
//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
func getEnvStatuses(env string) {
	log.Println("Got to getEnvStatuses")
	if debug {
		log.Println("Got to getEnvStatuses")
	}
	for i := 0; i < len(Services); i++ {
		if Services[i].Env == env {
			Services[i].Status = ""
			back := &backend.Local{}
			shell, err := ps.New(back)
			if err != nil {
				log.Println(err)
			}
			config := middleware.NewSessionConfig()
			config.ComputerName = Services[i].ComputerName
			if debug {
				log.Println(config)
			}

			defer shell.Exit()

			serviceName := Services[i].Service
			wmi := " Get-Service -Name \"" + serviceName + "\"  -ComputerName " + Services[i].ComputerName
			if debug {
				log.Println(wmi)
			}
			stdout, stderr, err := shell.Execute(wmi)
			if err != nil {
				log.Println(err)
			}
			if stderr != "" {
				log.Println(stderr)
			}
			if strings.Contains(stdout, "Running") {
				Services[i].Status = "up"
				log.Println("status up")
			} else if strings.Contains(stdout, "Stopped") {
				Services[i].Status = "dn"
				log.Println("status down")
			} else {
				log.Println("status unk")
				Services[i].Status = "ukn"
			}
			log.Println(Services[i].Status)
		}
	}
	log.Println("done with getEnvStatuses")
	log.Println("Services have been registered.")
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
//  This will start or stop an idividual service.  If the serive is donw it will bring it up
//  or if it is up it will bring is down.
//  Lastly, if the service state is unknown, it will try to bring it up.
//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
func startStopService(prodSel string) {
		log.Println("got to startStopService")
		log.Println(prodSel)

	a, err := strconv.Atoi(prodSel)
	if err != nil {
		log.Println("Ascii to int conversion error.")
	}
	if Services[a].Status == "up" {
		log.Println(prodSel + " is currently up.  stopping service")
		stopService(a)
	} else if Services[a].Status == "dn" {
		log.Println(prodSel + " is currently down.  starting service")
		startService(a)
	} else {
		log.Println(prodSel + " appears to be down but we could not connect to find out.  We will be starting it up anyway!!!")
		startService(a)
	}
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
//  Stops a service
//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
func stopService(id int) {
	if debug {
		log.Println(id)
		log.Println("got to stopService")
	}
	back := &backend.Local{}
	shell, err := ps.New(back)
	if err != nil {
		log.Println(err)
	}
	config := middleware.NewSessionConfig()
	config.ComputerName = Services[id].ComputerName
	if debug {
		log.Println(config)
	}
	defer shell.Exit()
	serviceName := Services[id].Service
	wmi := " Get-Service -Name \"" + serviceName + "\"  -ComputerName " + Services[id].ComputerName + " | Stop-Service -Force "
	if debug {
		log.Println(wmi)
	}

	stdout, stderr, err := shell.Execute(wmi)
	if err != nil {
		log.Println(err)
	}
	if stderr != "" {
		log.Println(stderr)
	}
	Services[id].Status = "dn"
	if debug {
		log.Println(stdout)
	}
	log.Println("stopService: " + Services[id].ComputerName + " " + Services[id].Service + "" + Services[id].Status)
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
//  Starts a service
//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
func startService(id int) {
	if debug {
		log.Println("go to startServce")
	}
	back := &backend.Local{}
	shell, err := ps.New(back)
	if err != nil {
		log.Println(err)
	}
	config := middleware.NewSessionConfig()
	config.ComputerName = Services[id].ComputerName
	if debug {
		log.Println("Config:")
		log.Println(config)
	}
	defer shell.Exit()
	serviceName := Services[id].Service
	if debug {
		log.Println("Service Name: " + serviceName)
	}
	wmi := " Get-Service -Name \"" + serviceName + "\"  -ComputerName " + Services[id].ComputerName + " | Start-Service "
	if debug {
		log.Println(wmi)
	}
	stdout, stderr, err := shell.Execute(wmi)
	if err != nil {
		log.Println(err)
	}
	if stderr != "" {
		log.Println(stderr)
	}
	if stdout != "" {
		log.Println(stdout)
	}
	getStatus(id)

	log.Println("startService: " + Services[id].ComputerName + " " + Services[id].Service + "" + Services[id].Status)
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
// this will get the status of a single service id
//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
func refreshSatus(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	log.Println("got to refreshStatus")
	// Once done we need to redirect back to the same page so
	//fmt.Println("got to refreshStatus")
	ids, err  := strconv.Atoi(  req.URL.Query().Get("id"))
	if err != nil{
		log.Println("Error ", err)
	}
	getStatus(ids)

	redirectTarget := URLName
	http.Redirect(w, req, redirectTarget, 302)
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
// Gets the status for an individual service
//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
func getStatus(id int) {
	if debug {
		log.Println(id)
		log.Println("got to stopService")
	}
	back := &backend.Local{}
	shell, err := ps.New(back)
	if err != nil {
		log.Println(err)
	}
	config := middleware.NewSessionConfig()
	config.ComputerName = Services[id].ComputerName

	if debug {
		log.Println(config)
	}
	defer shell.Exit()
	serviceName := Services[id].Service
	wmi := " Get-Service -Name \"" + serviceName + "\"  -ComputerName " + Services[id].ComputerName
	log.Println(wmi)
	stdout, stderr, err := shell.Execute(wmi)

	if err != nil {
		log.Println(err)
	}
	if stderr != "" {
		log.Println(stderr)
	}
	log.Println(stdout)
	if strings.Contains(stdout, "Running") {
		Services[id].Status = "up"
		log.Println("status up")
	} else if strings.Contains(stdout, "Stopped") {
		Services[id].Status = "dn"
		log.Println("status down")
	} else {
		log.Println("status unk")
		Services[id].Status = "ukn"
	}

	log.Println("GetStatus: " + Services[id].ComputerName + " " + Services[id].Service + "" + Services[id].Status)
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
//  Displays the inital login page.  Accesses by entering /
//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
func indexPageHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	log.Println("got to indexPageHandler")
	fmt.Fprintf(w, indexPage)
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
//  Checks logon and processes page to either go back (invalid logon) or go forward /home
//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
func logonHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	log.Println("got to logonHandler")
	name := r.FormValue("name")
	pass := r.FormValue("password")
	log.Println(name)

	redirectTarget := "/"
	if name != "" && pass != "" {
		// .. check credentials ..
		var msg string
		log.Println(msg)
		log.Println("name ", name)

		//userOK := validateUser(name, pass)
		userOK := true
		fmt.Println("userOK bool status = ", strconv.FormatBool(userOK))

		if userOK {
			userName = name
			setSession(name, w)
			redirectTarget = "/home"
			http.Redirect(w, r, redirectTarget, 302)
		} else {
			log.Println(err)
			redirectTarget = "/"
			http.Redirect(w, r, redirectTarget, 302)
		}
	}
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
//  Function to validate user in LDAP
//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
func validateUser(name string, pass string) bool {
	log.Println("got to validateUser function")
	ok := false
	log.Println("OK Value1= ", ok)
	//ldap.url=ldap://daylight.ads:389
	//ldap.username=CN=SVC_SSCAP,OU=Production,OU=IT Service Account,OU=Information Technology,OU=Daylight Users,DC=daylight,DC=ads
	//ldap.pword=$$c@p@cc0unT

	bindusername := "CN=SVC_SSCAP,OU=Production,OU=IT Service Account,OU=Information Technology,OU=Daylight Users,DC=daylight,DC=ads"
	bindpassword := "$$c@p@cc0unT"

	conn, err := ldap.Dial("tcp", "daylight.ads:389")

	log.Println("Connection Details:")
	log.Println(conn)
	if err != nil {
		log.Println("ldap.Dial Error= ", err)
	}
	defer conn.Close()

	err = conn.Bind(bindusername, bindpassword)
	if err != nil {
		log.Println("conn.Bind Error= ", err)
	}else{
		log.Println("Conn after bind: = ")
		log.Println(conn)
	}
	log.Println(fmt.Sprintf("(&(objectClass=organizationalPerson)(uid=%s))", name))
	// Search for the given username
	searchRequest := ldap.NewSearchRequest(
		"DC=daylight,DC=ads",
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(&(objectClass=organizationalPerson)(uid=%s))", name),
		[]string{"dn", "sAMAccountName", "mail", "sn", "givenName"},
		nil,
	)
	log.Println(searchRequest)
	sr, err := conn.Search(searchRequest)
	if err != nil {
		log.Println("conn.Search Error= ", err)
	}
	log.Println("searchRequest")
	log.Println(searchRequest)
	log.Println(sr.Entries)
	log.Println(err)

	if len(sr.Entries) != 1 {
		log.Println("User does not exist or too many entries returned")
		log.Println("sr.Entries: = ")
		log.Println(sr.Entries)
		ok = false
	} else {

		userdn := sr.Entries[0].DN
		log.Println("userdn entry= ", userdn)
		if len(userdn) > 0 {

			// Bind as the user to verify their password
			err = conn.Bind(userdn, pass)
			if err != nil {
				log.Println("conn.Bind Error=", err)
			}
			ok = true
		} else {
			ok = false
		}
	}

	log.Println("OK Value2= ", ok)
	return ok
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
//  Sets session cookie
//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
func setSession(userName string, response http.ResponseWriter) {
	log.Println("got to setSession")
	value := map[string]string{
		"name": userName,
	}
	if encoded, err := cookieHandler.Encode("session", value); err == nil {
		cookie := &http.Cookie{
			Name:  "session",
			Value: encoded,
			Path:  "/",
		}
		http.SetCookie(response, cookie)
	}
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
//  Gets user by cookie id for returnsgo
//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
func getUserName(request *http.Request) (userName string) {
	log.Println("got to getUserName")
	if cookie, err := request.Cookie("session"); err == nil {
		cookieValue := make(map[string]string)
		if err = cookieHandler.Decode("session", cookie.Value, &cookieValue); err == nil {
			userName = cookieValue["name"]
		}
	}
	return userName
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
//  Clears session. Caused by logging out.  (not yet impelmented)
//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
func clearSession(response http.ResponseWriter) {
	log.Println("got to clearSession")
	cookie := &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(response, cookie)
}


//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
//  Web Socket impelmentation
//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
//func handleConnections(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
//	// Upgrade initial GET request to a websocket
//	ws, err := upgrader.Upgrade(w, r, nil)
//	if err != nil {
//		log.Fatal(err)
//	}
//	// Make sure we close the connection when the function returns
//	defer ws.Close()
//	for {
//		var msg Message
//
//		if err != nil {
//			log.Println(err)
//			fmt.Println(err)
//			delete(clients, ws)
//			break
//		}
//		broadcast <- msg
//	}
//	// Register our new client
//	clients[ws] = true
//}
//func handleConnections(w http.ResponseWriter, r *http.Request, _ httprouter.Params){
//	ws, err := upgrader.Upgrade(w, r, nil)
//	if err != nil {
//		fmt.Println(err)
//		return
//	}
//	defer ws.Close()
//	clients[ws] = true
//
//	for {
//		msgType, msg, err := conn.ReadMessage()
//		fmt.Println(msgType)
//		fmt.Println(string(msg))
//		if err != nil {
//			fmt.Println(err)
//			return
//		}
//
//		if string(msg) == "ping" {
//			fmt.Println("ping")
//			time.Sleep(2 * time.Second)
//			err = conn.WriteMessage(msgType, []byte("pong"))
//			if err != nil {
//				fmt.Println(err)
//				return
//			}
//		} else {
//			fmt.Println(string(msg))
//			conn.Close()
//			return
//		}
//	}
//
//}
//func handleMessages(message Message) {
//	for{
//		msg := <-broadcast
//		byteArray := []byte(msg.env)
//		fmt.Println(msg)
//		fmt.Println(byteArray)
//		for client := range clients{
//			err = client.WriteMessage(1, byteArray )
//			if err != nil{
//				log.Println("error: %v" , err)
//				fmt.Println( "error: %v" , err)
//				client.Close()
//				delete(clients, client)
//			}
//		}
//	}
//}
//

