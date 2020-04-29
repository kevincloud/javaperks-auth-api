package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	ldap "gopkg.in/ldap.v3"

	vault "github.com/hashicorp/vault/api"
	"github.com/julienschmidt/httprouter"
	"github.com/mitchellh/mapstructure"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// API : Service configuration
type API struct {
	Port         string
	VaultAddr    string
	VaultToken   string
	LdapHost     string
	LdapAdmin    string
	LdapPassword string
	Localhost    bool
}

// User : The user data to be returned to the application
type User struct {
	Username   string `json:"username"`
	Customerno string `json:"customerno"`
	Message    string `json:"message"`
	Success    bool   `json:"success"`
	Error      error  `json:"error"`
}

// VaultData : makes json payload usable
type VaultData struct {
	Data struct {
		Customerno string `json:"customerno"`
		Password   string `json:"password"`
		Username   string `json:"username"`
	} `json:"data"`
	Metadata struct {
		CreatedTime  time.Time `json:"created_time"`
		DeletionTime string    `json:"deletion_time"`
		Destroyed    bool      `json:"destroyed"`
		Version      int       `json:"version"`
	} `json:"metadata"`
}

// Authenticate : authenticate directly with ldap
func (api API) Authenticate(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Content-Type", "text/plain")

	defer r.Body.Close()

	r.ParseForm()

	username := r.Form.Get("username")
	password := r.Form.Get("password")

	bindusername := api.LdapAdmin
	bindpassword := api.LdapPassword

	l, err := ldap.Dial("tcp", fmt.Sprintf("%s:%d", api.LdapHost, 389))
	if err != nil {
		j, _ := json.Marshal(User{
			Username:   "",
			Customerno: "",
			Message:    "Couldn't connect to OpenLDAP",
			Success:    false,
			Error:      err,
		})
		fmt.Fprint(w, string(j))
		return
	}
	defer l.Close()

	err = l.Bind(bindusername, bindpassword)
	if err != nil {
		j, _ := json.Marshal(User{
			Username:   "",
			Customerno: "",
			Message:    "Bad bind credentials",
			Success:    false,
			Error:      err,
		})
		fmt.Fprint(w, string(j))
		return
	}

	searchRequest := ldap.NewSearchRequest(
		"dc=javaperks,dc=local",
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(&(objectClass=inetOrgPerson)(uid=%s))", username),
		[]string{"dn", "uid", "employeeNumber"},
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		j, _ := json.Marshal(User{
			Username:   "",
			Customerno: "",
			Message:    "There was an error searching the directory",
			Success:    false,
			Error:      err,
		})
		fmt.Fprint(w, string(j))
		return
	}

	if len(sr.Entries) != 1 {
		j, _ := json.Marshal(User{
			Username:   "",
			Customerno: "",
			Message:    "Bad username/password",
			Success:    false,
			Error:      err,
		})
		fmt.Fprint(w, string(j))
		return
	}

	userdn := sr.Entries[0].DN
	userid := sr.Entries[0].GetAttributeValue("uid")
	custno := sr.Entries[0].GetAttributeValue("employeeNumber")

	err = l.Bind(userdn, password)
	if err != nil {
		j, _ := json.Marshal(User{
			Username:   userid,
			Customerno: custno,
			Message:    "Bad username/password",
			Success:    false,
			Error:      err,
		})
		fmt.Fprint(w, string(j))
		return
	}

	err = l.Bind(bindusername, bindpassword)
	if err != nil {
		j, _ := json.Marshal(User{
			Username:   userid,
			Customerno: custno,
			Message:    "Error completing process",
			Success:    false,
			Error:      err,
		})
		fmt.Fprint(w, string(j))
		return
	}

	j, _ := json.Marshal(User{
		Username:   sr.Entries[0].GetAttributeValue("uid"),
		Customerno: sr.Entries[0].GetAttributeValue("employeeNumber"),
		Message:    "User successfully authenticated",
		Success:    true,
		Error:      err,
	})
	fmt.Fprint(w, string(j))
	return
}

// // Authenticate3 : authenticate using vault auth ldap
// func (api API) Authenticate3(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
// 	w.Header().Set("Access-Control-Allow-Origin", "*")
// 	w.Header().Set("Access-Control-Allow-Methods", "POST")
// 	w.Header().Set("Content-Type", "text/plain")

// 	v := VaultData{}

// 	defer r.Body.Close()

// 	r.ParseForm()

// 	username := r.Form.Get("username")
// 	password := r.Form.Get("password")
// 	vaultpath := fmt.Sprintf("auth/ldap/login/%s", username)

// 	client, err := vault.NewClient(&vault.Config{Address: api.VaultAddr, HttpClient: httpClient})
// 	if err != nil {
// 		j, _ := json.Marshal(User{
// 			Username:   "",
// 			Customerno: "",
// 			Message:    "Couldn't connect to Vault",
// 			Success:    false,
// 			Error:      err,
// 		})
// 		fmt.Fprint(w, string(j))
// 		return
// 	}
// 	client.SetToken(api.VaultToken)
// 	client.Auth
// 	data, err := client.Logical().Read(vaultpath)
// 	if err != nil {
// 		j, _ := json.Marshal(User{
// 			Username:   "",
// 			Customerno: "",
// 			Message:    "Vault secret path not found",
// 			Success:    false,
// 			Error:      err,
// 		})
// 		fmt.Fprint(w, string(j))
// 		return
// 	}

// 	mapstructure.Decode(data.Data, &v)

// 	if v.Data.Password == password {
// 		j, _ := json.Marshal(User{
// 			Username:   v.Data.Username,
// 			Customerno: v.Data.Customerno,
// 			Message:    "Authentication Successful",
// 			Success:    true,
// 			Error:      nil,
// 		})
// 		fmt.Fprint(w, string(j))
// 	} else {
// 		j, _ := json.Marshal(User{
// 			Username:   "",
// 			Customerno: "",
// 			Message:    "Bad password",
// 			Success:    false,
// 			Error:      nil,
// 		})
// 		fmt.Fprint(w, string(j))
// 	}
// }

// Authenticate2 : authenticate the username/pass
func (api API) Authenticate2(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Content-Type", "text/plain")

	v := VaultData{}

	defer r.Body.Close()

	r.ParseForm()

	username := r.Form.Get("username")
	password := r.Form.Get("password")
	vaultpath := fmt.Sprintf("usercreds/data/%s", username)

	client, err := vault.NewClient(&vault.Config{Address: api.VaultAddr, HttpClient: httpClient})
	if err != nil {
		j, _ := json.Marshal(User{
			Username:   "",
			Customerno: "",
			Message:    "Couldn't connect to Vault",
			Success:    false,
			Error:      err,
		})
		fmt.Fprint(w, string(j))
		return
	}
	client.SetToken(api.VaultToken)
	data, err := client.Logical().Read(vaultpath)
	if err != nil {
		j, _ := json.Marshal(User{
			Username:   "",
			Customerno: "",
			Message:    "Vault secret path not found",
			Success:    false,
			Error:      err,
		})
		fmt.Fprint(w, string(j))
		return
	}

	mapstructure.Decode(data.Data, &v)

	if v.Data.Password == password {
		j, _ := json.Marshal(User{
			Username:   v.Data.Username,
			Customerno: v.Data.Customerno,
			Message:    "Authentication Successful",
			Success:    true,
			Error:      nil,
		})
		fmt.Fprint(w, string(j))
	} else {
		j, _ := json.Marshal(User{
			Username:   "",
			Customerno: "",
			Message:    "Bad password",
			Success:    false,
			Error:      nil,
		})
		fmt.Fprint(w, string(j))
	}
}

// Run : Launch the thing
func (api API) Run() {
	ipaddr := "0.0.0.0"
	if api.Localhost {
		ipaddr = "127.0.0.1"
	}
	apiPort := fmt.Sprintf("%s:%s", ipaddr, api.Port)
	router := httprouter.New()

	router.POST("/auth", api.Authenticate)
	log.Fatal(http.ListenAndServe(apiPort, router))
}

// New : Server setup
func New(port string) *API {
	localhost := false
	_, localhost = os.LookupEnv("LOCALHOST_ONLY")

	api := &API{
		Port:         port,
		VaultAddr:    os.Getenv("VAULT_ADDR"),
		VaultToken:   os.Getenv("VAULT_TOKEN"),
		LdapHost:     os.Getenv("LDAP_HOST"),
		LdapAdmin:    os.Getenv("LDAP_ADMIN"),
		LdapPassword: os.Getenv("LDAP_PASSWORD"),
		Localhost:    localhost,
	}

	return api
}
