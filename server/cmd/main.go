package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/LucianoGiope/labsCloudrun/configs"
	"github.com/LucianoGiope/labsCloudrun/pkg/httpResponseErr"
)

type Address struct {
	Cep    string `json:"cep"`
	Rua    string `json:"logradouro"`
	Bairro string `json:"bairro"`
	Cidade string `json:"localidade"`
	Estado string `json:"uf"`
}
type WeatherResult struct {
	Location struct {
		Name    string `json:"name"`
		Country string `json:"country"`
	} `json:"location"`
	Current struct {
		TempC     float64 `json:"temp_c"`
		TempF     float64 `json:"temp_f"`
		TempK     float64 `json:"temp_k"`
		Condition struct {
			Text string `json:"text"`
		} `json:"condition"`
	} `json:"current"`
}

type WeatherApi struct {
	Cep    string  `json:"cep"`
	Rua    string  `json:"logradouro"`
	Bairro string  `json:"bairro"`
	Cidade string  `json:"localidade"`
	Estado string  `json:"uf"`
	TempC  float64 `json:"temp_c"`
	TempF  float64 `json:"temp_f"`
	TempK  float64 `json:"temp_k"`
}

func NewWeatherApi(add *Address, wr *WeatherResult) *WeatherApi {
	return &WeatherApi{
		add.Cep,
		add.Rua,
		add.Bairro,
		add.Cidade,
		add.Estado,
		wr.Current.TempC,
		wr.Current.TempF,
		wr.Current.TempK,
	}
}

/*
retorno da Api
"temp_c": 14.2,
"temp_f": 57.6,
*/

//  { "temp_C": 28.5, "temp_F": 28.5, "tempK": 28.5 }

func main() {
	println("\nIniciando o servidor na porta 8080 e aguardando requisições")

	routers := http.NewServeMux()

	routers.HandleFunc("/", searchCEPHandler)
	routers.HandleFunc("/weatherByCep/{cep}", searchCEPHandler)
	err := http.ListenAndServe(":8080", routers)
	if err != nil {
		log.Fatal(err)
	}

}

func searchCEPHandler(w http.ResponseWriter, r *http.Request) {

	var msgErro *httpResponseErr.SHttpError

	w.Header().Set("Content-Type", "application/json")

	config, err := configs.LoadConfig(".")
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		msgErro = httpResponseErr.NewHttpError(fmt.Sprintf("Error loading configuration: %v\n", err), http.StatusInternalServerError)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(msgErro)
		return
	}

	urlAccess := strings.Split(r.URL.Path, "/")[1]
	if urlAccess != "weatherByCep" {
		fmt.Printf("The access must by in  of the endpoint http://localhost:8080/weatherByCep. [ErrorCode:%d]\n", http.StatusBadRequest)
		msgErro = httpResponseErr.NewHttpError("The access must by in  of the endpoint http://localhost:8080/weatherByCep\n", http.StatusNotFound)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(msgErro)
		return
	}
	CepCurrency := r.PathValue("cep")
	if CepCurrency == "" {
		fmt.Printf("CEP currency not send in parameter. [ErrorCode:%d]\n", http.StatusBadRequest)
		msgErro = httpResponseErr.NewHttpError("CEP currency not send in parameter.\n Exemple: http://localhost:8080/weatherByCep/{CepCurrency}\n", http.StatusBadRequest)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(msgErro)
		return
	}

	regex := regexp.MustCompile(`[^0-9]+`)
	apenasNumeros := regex.ReplaceAllString(CepCurrency, "")
	if len(apenasNumeros) <= 0 {
		fmt.Printf("A zipcode must be entered, please try again. !! [ErrorCode:%d]\n", http.StatusBadRequest)
		msgErro = httpResponseErr.NewHttpError("A zipcode must be entered, please try again. !!", http.StatusBadRequest)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(msgErro)
		return
	}
	if len(apenasNumeros) != 8 {
		fmt.Printf("The zipcode %s is not a valid number, please try again. !! [ErrorCode:%d]\n", CepCurrency, http.StatusUnprocessableEntity)
		msgErro = httpResponseErr.NewHttpError(fmt.Sprintf("The zipcode %s is not a valid number, please try again. !!", CepCurrency), http.StatusUnprocessableEntity)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(msgErro)
		return
	}

	ctxClient := r.Context()

	timeAtual := time.Now()
	fmt.Printf("\n-> Searching local climate for the ZIPCODE:%s in %v.\n", CepCurrency, timeAtual.Format("02/01/2006 15:04:05 ")+timeAtual.String()[20:29]+" ms")

	errCode := 0
	errText := ""
	var viacepResult Address

	ctxSearchViaCep, cancelSearchVC := context.WithTimeout(ctxClient, time.Second*1)
	defer cancelSearchVC()
	urlSearch := fmt.Sprintf(config.UrlCep, CepCurrency)

	nomeCidade := ""
	// Search data cep
	resBodyCep, err := searchCep(ctxSearchViaCep, urlSearch, "ViaCep")
	if err != nil {
		msgErrFix := "__Error searching for ViaCep."
		if ctxSearchViaCep.Err() != nil {
			errCode = http.StatusRequestTimeout
			errText = msgErrFix + "\n____[MESSAGE] Search time exceeded."
		} else {
			errCode = http.StatusBadRequest
			errText = msgErrFix + "\n____[MESSAGE] Request failed."
		}
		msgErro = httpResponseErr.NewHttpError(errText, errCode)
		w.WriteHeader(errCode)
		json.NewEncoder(w).Encode(msgErro)

	} else if resBodyCep != nil {
		// var viacepResult Address
		err = json.Unmarshal(resBodyCep, &viacepResult)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nError converting response resBodyCep:%v\n", err.Error())
		}
		nomeCidade = viacepResult.Cidade
		if nomeCidade == "" {
			fmt.Printf("\n--> Unable to locate city for zipcode:%s. [ErrorCode:%d]\n", CepCurrency, http.StatusNotFound)
		} else {
			fmt.Printf("\n--> The city %s has been located.\n", viacepResult.Cidade)
		}
	}

	if nomeCidade != "" {
		ctxSearchWeather, cancelSearchWeather := context.WithTimeout(ctxClient, time.Second*3)
		defer cancelSearchWeather()

		urlWeather := fmt.Sprint(config.UrlWeather) + fmt.Sprint(config.APIKeyWeather)

		resBodyWeather, err := searchWeather(ctxSearchWeather, urlWeather, nomeCidade)
		if err != nil {
			msgErrFix := "__Error searching for WeatherApi."
			if ctxSearchWeather.Err() != nil {
				errCode = http.StatusRequestTimeout
				errText = msgErrFix + "\n____[MESSAGE] Search time exceeded."
			} else {
				errCode = http.StatusBadRequest
				errText = msgErrFix + "\n____[MESSAGE] Request failed."
			}

			msgErro = httpResponseErr.NewHttpError(errText, errCode)
			w.WriteHeader(errCode)
			json.NewEncoder(w).Encode(msgErro)

		} else if resBodyWeather != nil {
			var weatherResult WeatherResult
			err = json.Unmarshal(resBodyWeather, &weatherResult)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nError converting response:%v\n", err.Error())
			}

			weatherResult.Current.TempK = weatherResult.Current.TempC + 273
			fmt.Printf("\n--> Valores coletados para cidade: %s com temperaturas \n_______Celsius:%v \n_______Fahrenheit:%v \n_______Kelvin:%v \n_______Tempo:%s \n",
				weatherResult.Location.Name,
				weatherResult.Current.TempC,
				weatherResult.Current.TempF,
				weatherResult.Current.TempK,
				weatherResult.Current.Condition.Text)

			nwa := NewWeatherApi(&viacepResult, &weatherResult)

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(nwa)
		}
	} else {
		errCode = http.StatusNotFound
		errText = "Error searching for ViaCep.\n____[MESSAGE] Não foi possível localizar a cidade para o CEP informado."
		msgErro = httpResponseErr.NewHttpError(errText, errCode)
		w.WriteHeader(errCode)
		json.NewEncoder(w).Encode(msgErro)

	}

	fmt.Printf("\n-> Time total in milliseconds traveled %v.\n\n", time.Since(timeAtual))
}

func searchCep(ctx context.Context, urlSearch, apiName string) ([]byte, error) {

	timeAtual := time.Now()
	fmt.Printf("\n--> Starting search type:%s in %s\n", apiName, timeAtual.Format("02/01/2006 15:04:05 ")+timeAtual.String()[20:29]+" ms")

	bodyResp, err := requestApi(ctx, urlSearch)

	select {
	case <-ctx.Done():
		err2 := ctx.Err()
		if err2 == context.Canceled {
			fmt.Printf("\n________Cancelled to consult with supplier %s\n", apiName)
		} else if err2 == context.DeadlineExceeded {
			timeAtual = time.Now()
			fmt.Printf("\n________Time exceeded to consult with supplier %s in %s\n", apiName, timeAtual.Format("02/01/2006 15:04:05 ")+timeAtual.String()[20:29]+" ms")
		} else {
			fmt.Printf("\n________Query on %s abandoned for unknown reason.\n [ERROR] %v\n", apiName, err)
		}
		return nil, nil
	default:
		if err != nil {
			fmt.Printf("\n________Failed to query Zipcode in %s\n [MESSAGE]%v\n", apiName, err.Error())
			return nil, err
		} else {
			timeAtual = time.Now()
			fmt.Printf("\n________Captured Zipcode data with API: %s in %s\n", apiName, timeAtual.Format("02/01/2006 15:04:05 ")+timeAtual.String()[20:29]+" ms")

			return *bodyResp, nil
		}
	}
}
func removeAccents(texto string) string {
	acentuados := map[rune]rune{
		'á': 'a', 'é': 'e', 'í': 'i', 'ó': 'o', 'ú': 'u',
		'à': 'a', 'è': 'e', 'ì': 'i', 'ò': 'o', 'ù': 'u',
		'ã': 'a', 'õ': 'o',
		'â': 'a', 'ê': 'e', 'î': 'i', 'ô': 'o', 'û': 'u',
		'ç': 'c',
	}

	for r, v := range acentuados {
		texto = strings.ReplaceAll(texto, string(r), string(v))
	}
	texto = strings.ReplaceAll(texto, " ", "%20")

	return texto
}
func searchWeather(ctx context.Context, urlWeather, cidade string) ([]byte, error) {

	timeAtual := time.Now()
	fmt.Printf("\n--> Starting city weather search %s in %s\n", cidade, timeAtual.Format("02/01/2006 15:04:05 ")+timeAtual.String()[20:29]+" ms")
	cidadeSend := removeAccents(cidade)
	bodyResp, err := requestApi(ctx, urlWeather+"&q="+cidadeSend)
	select {
	case <-ctx.Done():
		err2 := ctx.Err()
		if err2 == context.Canceled {
			fmt.Printf("\n________Cancelled weather consultation for %s\n", cidade)
		} else if err2 == context.DeadlineExceeded {
			timeAtual = time.Now()
			fmt.Printf("\n________Timeout to check weather %s in %s\n", cidade, timeAtual.Format("02/01/2006 15:04:05 ")+timeAtual.String()[20:29]+" ms")
		} else {
			fmt.Printf("\n________Query for %s abandoned for unknown reason.\n [ERROR] %v\n", cidade, err)
		}
		return nil, nil
	default:
		if err != nil {
			fmt.Printf("\n________Failed to query weather in %s\n [MESSAGE]%v\n", cidade, err.Error())
			return nil, err
		} else {
			timeAtual = time.Now()
			fmt.Printf("\n________Weather data captured for city: %s on %s\n", cidade, timeAtual.Format("02/01/2006 15:04:05 ")+timeAtual.String()[20:29]+" ms")

			return *bodyResp, nil
		}
	}
}

func requestApi(ctx context.Context, urlSearch string) (*[]byte, error) {

	req, err := http.NewRequestWithContext(ctx, "GET", urlSearch, nil)
	if err != nil {
		return nil, err
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("query failed with code:%v", response.Status)
	}

	bodyResp, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf(" error reading response:%v", err.Error())
	}

	return &bodyResp, nil
}
