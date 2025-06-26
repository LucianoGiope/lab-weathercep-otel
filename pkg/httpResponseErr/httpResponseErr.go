package httpResponseErr

import (
	"encoding/json"
	"errors"
	"fmt"
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
		return fmt.Sprintf("%s ErrorCode:%d", dataResult.Message, dataResult.Code), errors.New(err.Error())
	}
	return fmt.Sprintf("%s ErrorCode:%d", dataResult.Message, dataResult.Code), nil
}
