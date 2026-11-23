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
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://tyradex.app/api/v1/pokemon"

var _temp = template.Must(template.ParseGlob("./templates/*.html"))

type Pokemon struct {
	Id  int    `json:"pokedex_id"`
	Generation int    `json:"generation"`
	Category   string `json:"category"`
	Name       struct {
		Fr string `json:"fr"`
		En string `json:"en"`
		Jp string `json:"jp"`
	} `json:"name"`
	Sprites struct {
		Regular string `json:"regular"`
	} `json:"sprites"`
	Types []Type `json:"types"`
	Stats Stats  `json:"stats"`
}

type Type struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

type Stats struct {
	HP     int `json:"hp"`
	Atk    int `json:"atk"`
	Def    int `json:"def"`
	SpeAtk int `json:"spe_atk"`
	SpeDef int `json:"spe_def"`
	Vit    int `json:"vit"`
}

var listPokemon []Pokemon

var (
	ErrRequestCreation = errors.New("impossible de créer la requête")
	ErrRequestSend     = errors.New("impossible d'envoyer la requête")
	ErrNotFound        = errors.New("ressource introuvable")
	ErrInvalidID       = errors.New("identifiant non valide")
	ErrUnexpectedState = errors.New("réponse inattendue de l'API")
	ErrDecode          = errors.New("impossible de décoder la réponse")
)

func getCharacter(client *http.Client, id string) (*Pokemon, error) {

	endpoint := fmt.Sprintf("%s/pokemon/%d", baseURL, id)

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

	var pokemon Pokemon
	if err := json.NewDecoder(response.Body).Decode(&pokemon); err != nil {
		return nil, ErrDecode
	}

	return &pokemon, nil
}

func main() {

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	
	jErr := loadPokemon()
	if jErr != nil {
		log.Fatalf("Erreur chargement JSON : %v", jErr)
	}

	router := http.NewServeMux()

	router.HandleFunc("/{$}", HomeHandler)
	router.HandleFunc("/pokemon", PokemonHandler)
	router.HandleFunc("/pokemon/{id}", PokemonByIdHandler)
	router.HandleFunc("/", NotfoundHandler)

	fileServer := http.FileServer(http.Dir("./assets"))
	router.Handle("/static/", http.StripPrefix("/static/", fileServer))

	err := http.ListenAndServe("localhost:8080", router)
	if err != nil {
		log.Fatalf("Erreur (Serveur) : Impossible de lancer le serveur : %v", err)
		return
	}

	id := "0"

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
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func loadPokemon() error {
    file, err := os.Open("pokedex.json")
    if err != nil {
        return err
    }
    defer file.Close()

    err = json.NewDecoder(file).Decode(&listPokemon)
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

func PokemonHandler(w http.ResponseWriter, r *http.Request) {
	result := listPokemon

	if name := strings.TrimSpace(r.URL.Query().Get("name")); name != "" {
		result = nil
		for _, p := range listPokemon {
			if strings.Contains(strings.ToLower(p.Name.Fr), strings.ToLower(name)) {
				result = append(result, p)
			}
		}
	}

	renderTemplate(w, "pokemon", result)
}

func PokemonByIdHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, "ID invalide", http.StatusBadRequest)
		return
	}

	for _, p := range listPokemon {
		if p.Id == id {
			renderTemplate(w, "pokemon-detail", p)
			return
		}
	}

	http.Error(w, "Pokémon introuvable", http.StatusNotFound)
}

func NotfoundHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Request : %s %s", r.Method, r.URL.Path)

	renderTemplate(w, "error", nil)
}

