package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/LucianoGiope/openTelemetry/configs"
	"github.com/LucianoGiope/openTelemetry/search-weather/pkg/httpResponseErr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

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
	TempC float64 `json:"temp_c"`
	TempF float64 `json:"temp_f"`
	TempK float64 `json:"temp_k"`
}

func NewWeatherApi(wr *WeatherResult) *WeatherApi {
	return &WeatherApi{
		wr.Current.TempC,
		wr.Current.TempF,
		wr.Current.TempK,
	}
}

func CreateNewServer() *http.ServeMux {

	routers := http.NewServeMux()
	routers.HandleFunc("/{cidade}", SearchWeatherHandler)

	return routers
}
func SearchWeatherHandler(w http.ResponseWriter, r *http.Request) {

	carrier := propagation.HeaderCarrier(r.Header)
	ctxClient := r.Context()
	ctxClient = otel.GetTextMapPropagator().Extract(ctxClient, carrier)
	tracer := otel.Tracer("search-weather")
	ctxClient, span := tracer.Start(ctxClient, "SearchWeatherHandler")
	defer span.End()

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

	nomeCidade := r.PathValue("cidade")
	if nomeCidade == "" {
		fmt.Printf("Nome Cidade currency not send in parameter. [ErrorCode:%d]\n", http.StatusBadRequest)
		msgErro = httpResponseErr.NewHttpError("Nome Cidade currency not send in parameter.\n Exemple: http://localhost:8080/{nomeCidade}\n", http.StatusBadRequest)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(msgErro)
		return
	}

	timeAtual := time.Now()
	fmt.Printf("\n-> Searching local climate for the CIDADE:%s in %v.\n", nomeCidade, timeAtual.Format("02/01/2006 15:04:05 ")+timeAtual.String()[20:29]+" ms")

	errCode := 0
	errText := ""

	if nomeCidade != "" {
		urlWeather := fmt.Sprint(config.UrlWeather) + fmt.Sprint(config.APIKeyWeather)

		resBodyWeather, err := searchWeather(ctxClient, urlWeather, nomeCidade)
		if err != nil {
			msgErrFix := "__Error searching for WeatherApi."
			if ctxClient.Err() != nil {
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

			nwa := NewWeatherApi(&weatherResult)

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(nwa)
		}
	} else {
		errCode = http.StatusNotFound
		errText = "Error searching Weather.\n____[MESSAGE] Nome da cidade não foi informada para consulta."
		msgErro = httpResponseErr.NewHttpError(errText, errCode)
		w.WriteHeader(errCode)
		json.NewEncoder(w).Encode(msgErro)

	}

	fmt.Printf("\n-> Time total in milliseconds traveled %v.\n\n", time.Since(timeAtual))
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
