package handlers

import (
	"encoding/json"
	"net/http"

	routing_api "github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api/metrics"
	"github.com/pivotal-golang/lager"
)

func handleProcessRequestError(w http.ResponseWriter, procErr error, log lager.Logger) {
	log.Error("error", procErr)
	err := routing_api.NewError(routing_api.ProcessRequestError, "Cannot process request: "+procErr.Error())
	retErr := marshalRoutingApiError(err, log)

	w.WriteHeader(http.StatusBadRequest)
	w.Write(retErr)
}

func handleNotFoundError(w http.ResponseWriter, err error, log lager.Logger) {
	log.Error("error", err)
	retErr := marshalRoutingApiError(routing_api.NewError(routing_api.ResourceNotFoundError, err.Error()), log)

	w.WriteHeader(http.StatusNotFound)
	w.Write(retErr)
}

func handleApiError(w http.ResponseWriter, apiErr *routing_api.Error, log lager.Logger) {
	log.Error("error", apiErr)
	retErr := marshalRoutingApiError(*apiErr, log)

	w.WriteHeader(http.StatusBadRequest)
	w.Write(retErr)
}

func handleDBCommunicationError(w http.ResponseWriter, err error, log lager.Logger) {
	log.Error("error", err)
	retErr := marshalRoutingApiError(routing_api.NewError(routing_api.DBCommunicationError, err.Error()), log)

	w.WriteHeader(http.StatusInternalServerError)
	w.Write(retErr)
}

func handleUnauthorizedError(w http.ResponseWriter, err error, log lager.Logger) {
	log.Error("error", err)

	retErr := marshalRoutingApiError(routing_api.NewError(routing_api.UnauthorizedError, err.Error()), log)
	metrics.IncrementTokenError()

	w.WriteHeader(http.StatusUnauthorized)
	w.Write(retErr)
}

func handleDBConflictError(w http.ResponseWriter, err error, log lager.Logger) {
	log.Error("error", err)
	retErr := marshalRoutingApiError(routing_api.NewError(routing_api.DBConflictError, err.Error()), log)

	w.WriteHeader(http.StatusConflict)
	w.Write(retErr)
}

func marshalRoutingApiError(err routing_api.Error, log lager.Logger) []byte {
	retErr, jsonErr := json.Marshal(err)
	if jsonErr != nil {
		log.Error("could-not-marshal-json", jsonErr)
	}

	return retErr
}
