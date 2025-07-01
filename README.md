
Ainda falta fazer:
- buildar o docker para ver se funciona os apps pelo docker. [OK]
- Implementar a telemetria o zipkin.




## SEGUNDO DESAFIO DO LABORATÓRIO proposto pelo curso.

## Texto descritivo do desafio  

**Objetivo:** Desenvolver um sistema em Go que receba um CEP, identifica a cidade e retorna o clima atual (temperatura em graus celsius, fahrenheit e kelvin) juntamente com a cidade. Esse sistema deverá implementar OTEL(Open Telemetry) e Zipkin.  

> Basedo no cenário conhecido "Sistema de temperatura por CEP" denominado Serviço B, será incluso um novo projeto, denominado Serviço A.  

> Requisitos - **Serviço A** (responsável pelo input):  
- O sistema deve receber um input de 8 dígitos via POST, através do schema:  { "cep": "29902555" }.
- O sistema deve validar se o input é valido (contem 8 dígitos) e é uma STRING.
- Caso seja válido, será encaminhado para o Serviço B via HTTP.
- Caso não seja válido, deve retornar:
- - Código HTTP: 422
- - Mensagem: invalid zipcode  

> Requisitos - **Serviço B** (responsável pela orquestração):  
- O sistema deve receber um CEP válido de 8 digitos
- O sistema deve realizar a pesquisa do CEP e encontrar o nome da localização, a partir disso, deverá retornar as temperaturas e formata-lás em: Celsius, Fahrenheit, Kelvin juntamente com o nome da localização.
- O sistema deve responder adequadamente nos seguintes cenários:
- Em caso de sucesso:
- - Código HTTP: 200
- - Response Body: { "city: "São Paulo", "temp_C": 28.5, "temp_F": 28.5, "temp_K": 28.5 }
- Em caso de falha, caso o CEP não seja válido (com formato correto):
- - Código HTTP: 422
- - Mensagem: invalid zipcode
- ​​​Em caso de falha, caso o CEP não seja encontrado:
- - Código HTTP: 404
- - Mensagem: can not find zipcode

> Após a implementação dos serviços, adicione a implementação do OTEL + Zipkin:  
- Implementar tracing distribuído entre Serviço A - Serviço B
- Utilizar span para medir o tempo de resposta do serviço de busca de CEP e busca de temperatura 

**Dicas:**  
- Utilize a API viaCEP (ou similar) para encontrar a localização que deseja consultar a temperatura: https://viacep.com.br/
- Utilize a API WeatherAPI (ou similar) para consultar as temperaturas desejadas: https://www.weatherapi.com/
- Para realizar a conversão de Celsius para Fahrenheit, utilize a seguinte fórmula: F = C * 1,8 + 32
- Para realizar a conversão de Celsius para Kelvin, utilize a seguinte fórmula: K = C + 273
- - Sendo F = Fahrenheit
- - Sendo C = Celsius
- - Sendo K = Kelvin  

Entrega:  
- O código-fonte completo da implementação.  
- Documentação explicando como rodar o projeto em ambiente dev.
- Utilize docker/docker-compose para que possamos realizar os testes de sua aplicação.

### Como utilizar a API

## Passo 1- Configurar as variáveis
- Adicionar seus dados validação e acessos à Api
- Acesse o arquivo .env na pasta server/cmd e altere a variável:
- - APIKEYWEATHER={SEU TOKEN PARA A API WEATHER}

## Passo 2- Deve-se subir o server
- [Modo1] Acesse a pasta: server/cmd
- Execute o comando 
- - go run main.go
-
- [Modo2] Suba o container docker
- Entre na pasta root do projeto
- docker-compose up --build weather-cep
- 

## Com servidor ativado, executar a consulta
- Utilizando o cliente em Go
- - Acesse a pasta client/cmd e rode o comando abaixo
- - go run main.go {NÚMERO DO CEP}
-
- Utilizando serviço http
- - Abra o arquivo consulta.http na pasta api
- - Informe um cep válido e execute.
-
- Utilizando o navegador
- - http://localhost:8080/weatherByCep/{NÚMERO DO CEP}


