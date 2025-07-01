package entity

import (
	"github.com/google/uuid"
)

type ID = uuid.UUID

func NewUUID() string {
	return uuid.New().String()
}
