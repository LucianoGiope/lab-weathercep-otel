package httpResponseErr

import (
	"encoding/json"
	"errors"
)

type SHttpError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func NewHttpError(msg string, code int) *SHttpError {
	return &SHttpError{msg, code}
}

func (qe *SHttpError) DisplayMessage(jsonBody []byte) (string, error) {
	var dataResult = qe
	err := json.Unmarshal(jsonBody, &dataResult)
	if err != nil {
		return dataResult.Message, errors.New(err.Error())
	}
	return dataResult.Message, nil
}
