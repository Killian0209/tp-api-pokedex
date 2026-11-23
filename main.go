package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"
)

const baseURL = "https://tyradex.app/api/v1/pokemon"

var _temp = template.Must(template.ParseGlob("./templates/*.html"))

type Character struct {
    Id int `json:"id"`
    Name string `json:"name"`
    Origin Origin `json:"origin"`
    Location Location `json:"location"`
    Image string `json:"image"`
}

type Origin struct {
	Name string `json:"name"`
}

type Location struct {
	Name string `json:"name"`
}

var listCharacter []Character

var (
	ErrRequestCreation = errors.New("impossible de créer la requête")
	ErrRequestSend     = errors.New("impossible d'envoyer la requête")
	ErrNotFound        = errors.New("ressource introuvable")
	ErrInvalidID       = errors.New("identifiant non valide")
	ErrUnexpectedState = errors.New("réponse inattendue de l'API")
	ErrDecode          = errors.New("impossible de décoder la réponse")
)

func getCharacter(client *http.Client, id string) (*Character, error) {

	jErr := loadProjects()
	if jErr != nil {
		log.Fatalf("Erreur chargement JSON : %v", jErr)
	}

	router := http.NewServeMux()

		router.HandleFunc("/{$}", HomeHandler)




	endpoint := fmt.Sprintf("%s/%v", baseURL, id)

	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, ErrRequestCreation
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, ErrRequestSend
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:

	case http.StatusNotFound:
		return nil, ErrNotFound
	case http.StatusInternalServerError:
		return nil, ErrInvalidID
	default:
		return nil, ErrUnexpectedState
	}

	var character Character
	if err := json.NewDecoder(response.Body).Decode(&character); err != nil {
		return nil, ErrDecode
	}

	return &character, nil
}

func main() {

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	id := "fdfdvfdf"

	character, err := getCharacter(client, id)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			log.Fatalf("Personnage %q introuvable, vérifiez l'id.", id)
		case errors.Is(err, ErrInvalidID):
			log.Fatalf("L'id %q n'est pas valide (un nombre est attendu).", id)
		case errors.Is(err, ErrRequestCreation):
			log.Fatalf("Problème interne lors de la création de la requête.")
		case errors.Is(err, ErrRequestSend):
			log.Fatalf("Impossible de joindre l'API, vérifiez votre connexion.")
		case errors.Is(err, ErrDecode):
			log.Fatalf("La réponse de l'API est illisible.")
		case errors.Is(err, ErrUnexpectedState):
			log.Fatalf("L'API a renvoyé une réponse inattendue.")
		default:
			log.Fatalf("Erreur inconnue : %v", err)
		}
	}

	fmt.Println(*character)
}

func loadProjects() error {
    file, err := os.Open("data.json")
    if err != nil {
        return err
    }
    defer file.Close()

    err = json.NewDecoder(file).Decode(&listCharacter)
    return err
}

func renderTemplate(w http.ResponseWriter, name string, data interface{}) {

	buffer := bytes	.Buffer{}

	errRender := _temp.ExecuteTemplate(&buffer, name, data)
	if errRender != nil {
		log.Printf("Erreur (Template) : Impossible de rendre le template %s : %v", name, errRender)
		http.Error(w, "Erreur chargement de la page", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	buffer.WriteTo(w)
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Request : %s %s", r.Method, r.URL.Path)

	renderTemplate(w, "index", nil)
}
