package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	vault "github.com/hashicorp/vault/api"
	"github.com/julienschmidt/httprouter"
	"github.com/mitchellh/mapstructure"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// API : Service configuration
type API struct {
	Port       string
	VaultAddr  string
	VaultToken string
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

// Authenticate : authenticate the username/pass
func (api API) Authenticate(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
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
	apiPort := fmt.Sprintf(":%s", api.Port)
	router := httprouter.New()

	router.POST("/auth", api.Authenticate)
	log.Fatal(http.ListenAndServe(apiPort, router))
}

// New : Server setup
func New(port string) *API {
	api := &API{
		Port:       port,
		VaultAddr:  os.Getenv("VAULT_ADDR"),
		VaultToken: os.Getenv("VAULT_TOKEN"),
	}

	return api
}
