package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/pivotal-golang/lager"
)

type Error struct {
	Type    string `json:"name"`
	Message string `json:"message"`
}

func (err Error) Error() string {
	return err.Message
}

const (
	ProcessRequestError  = "ProcessRequestError"
	RouteInvalidError    = "RouteInvalidError"
	DBCommunicationError = "DBCommunicationError"
	UnauthorizedError    = "UnauthorizedError"
)

func handleProcessRequestError(w http.ResponseWriter, procErr error, log lager.Logger) {
	log.Error("error", procErr)

	retErr, _ := json.Marshal(Error{ProcessRequestError, "Cannot process request"})

	w.WriteHeader(http.StatusBadRequest)
	w.Write(retErr)
}

func handleApiError(w http.ResponseWriter, apiErr *Error, log lager.Logger) {
	log.Error("error", apiErr)

	retErr, _ := json.Marshal(apiErr)

	w.WriteHeader(http.StatusBadRequest)
	w.Write(retErr)
}

func handleDBError(w http.ResponseWriter, err error, log lager.Logger) {
	log.Error("error", err)

	retErr, _ := json.Marshal(Error{DBCommunicationError, err.Error()})

	w.WriteHeader(http.StatusInternalServerError)
	w.Write(retErr)
}

func handleUnauthorizedError(w http.ResponseWriter, err error, log lager.Logger) {
	log.Error("error", err)

	retErr, _ := json.Marshal(Error{UnauthorizedError, err.Error()})

	w.WriteHeader(http.StatusUnauthorized)
	w.Write(retErr)
}
