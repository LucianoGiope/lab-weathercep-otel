package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/LucianoGiope/openTelemetry/configs"
	"github.com/LucianoGiope/openTelemetry/search-weather/pkg/httpResponseErr"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Address struct {
	Cep    string `json:"cep"`
	Rua    string `json:"logradouro"`
	Bairro string `json:"bairro"`
	Cidade string `json:"localidade"`
	Estado string `json:"uf"`
}

type Weather struct {
	TempC float64 `json:"temp_c"`
	TempF float64 `json:"temp_f"`
	TempK float64 `json:"temp_k"`
}

type WeatherAndCep struct {
	Cep    string  `json:"cep"`
	Rua    string  `json:"logradouro"`
	Bairro string  `json:"bairro"`
	Cidade string  `json:"localidade"`
	Estado string  `json:"uf"`
	TempC  float64 `json:"temp_c"`
	TempF  float64 `json:"temp_f"`
	TempK  float64 `json:"temp_k"`
}

func NewWeatherAndCep(add *Address, wr *Weather) *WeatherAndCep {
	return &WeatherAndCep{
		add.Cep,
		add.Rua,
		add.Bairro,
		add.Cidade,
		add.Estado,
		wr.TempC,
		wr.TempF,
		wr.TempK,
	}
}
func CreateNewServer() *http.ServeMux {

	routers := http.NewServeMux()
	routers.HandleFunc("/", SearchCEPHandler)
	routers.HandleFunc("/weatherByCep/{cep}", SearchCEPHandler)
	routers.Handle("/metrics", promhttp.Handler())

	return routers
}
func SearchCEPHandler(w http.ResponseWriter, r *http.Request) {

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
		err = json.Unmarshal(resBodyCep, &viacepResult)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nError converting response resBodyCep:%v\n", err.Error())
		}
		nomeCidade = viacepResult.Cidade
		if nomeCidade == "" {
			fmt.Printf("\n--> Unable to locate city for zipcode:%s. [ErrorCode:%d]\n", CepCurrency, http.StatusNotFound)
		} else {
			fmt.Printf("\n--> The city %s has been located.\n", viacepResult.Cidade)

			ctxServerWeather, cancelServerWeather := context.WithTimeout(ctxClient, time.Second*1)
			defer cancelServerWeather()
			urlServerWeather := config.UrlServerWeather
			resBodyResult, err := executeServerWeather(ctxServerWeather, urlServerWeather, nomeCidade)
			if err != nil {
				msgErrFix := "__Error executing server weather."
				if ctxSearchViaCep.Err() != nil {
					errCode = http.StatusRequestTimeout
					errText = msgErrFix + "\n____[MESSAGE] Search server weather time exceeded."
				} else {
					errCode = http.StatusBadRequest
					// errText = msgErrFix + "\n____[MESSAGE] Request server weather failed."
					errText = msgErrFix + fmt.Sprintf("\nMensagem do erro %v;", err.Error())
				}
				msgErro = httpResponseErr.NewHttpError(errText, errCode)
				w.WriteHeader(errCode)
				json.NewEncoder(w).Encode(msgErro)

			} else if resBodyResult != nil {
				var weatherResult Weather
				err = json.Unmarshal(resBodyResult, &weatherResult)
				if err != nil {
					fmt.Fprintf(os.Stderr, "\nError converting response server weather:%v\n", err.Error())
				}
				nav := NewWeatherAndCep(&viacepResult, &weatherResult)
				fmt.Printf("\nSeguem os dados solicitados:\n"+
					"ENDEREÇO RETORNADO\n"+
					" Rua: %s\n Bairro: %s\n Cidade: %s\n Estado: %s\n CEP: %s\n\n"+
					"CLIMA NA LOCALIDADE: \n Temperatura em Celsios:%v\n"+
					" Temperatura em Fahrenheit:%v\n"+
					" Temperatura em Kelvin:%v\n",
					nav.Rua,
					nav.Bairro,
					nav.Cidade,
					nav.Estado,
					nav.Cep,
					nav.TempC,
					nav.TempF,
					nav.TempK,
				)
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(nav)

			}

		}
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
func executeServerWeather(ctx context.Context, urlServerWeather, cidade string) ([]byte, error) {

	timeAtual := time.Now()
	fmt.Printf("\n--> Execute ServerWeather in %s\n", timeAtual.Format("02/01/2006 15:04:05 ")+timeAtual.String()[20:29]+" ms")
	cidadeSend := removeAccents(cidade)
	bodyResp, err := requestApi(ctx, urlServerWeather+cidadeSend)
	select {
	case <-ctx.Done():
		err2 := ctx.Err()
		if err2 == context.Canceled {
			fmt.Printf("\n________Cancelled SERVER weather consultation for %s\n", cidade)
		} else if err2 == context.DeadlineExceeded {
			timeAtual = time.Now()
			fmt.Printf("\n________Timeout to check SERVER weather %s in %s\n", cidade, timeAtual.Format("02/01/2006 15:04:05 ")+timeAtual.String()[20:29]+" ms")
		} else {
			fmt.Printf("\n________Query of SERVER weather for %s abandoned for unknown reason.\n [ERROR] %v\n", cidade, err)
		}
		return nil, nil
	default:
		if err != nil {
			fmt.Printf("\n________Failed to query SERVER weather in %s\n [MESSAGE]%v\n", cidade, err.Error())
			return nil, err
		} else {
			timeAtual = time.Now()
			fmt.Printf("\n________SERVER Weather data response for city: %s on %s\n", cidade, timeAtual.Format("02/01/2006 15:04:05 ")+timeAtual.String()[20:29]+" ms")

			return *bodyResp, nil
		}
	}
}
