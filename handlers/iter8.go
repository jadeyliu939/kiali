package handlers

import (
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/kiali/kiali/prometheus"
)

func Iter8Status(w http.ResponseWriter, r *http.Request) {
	business, err := getBusiness(r)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Services initialization error: "+err.Error())
		return
	}
	iter8Info := business.Iter8.GetIter8Info()
	RespondWithJSON(w, http.StatusOK, iter8Info)
}

func Iter8Experiments(w http.ResponseWriter, r *http.Request) {
	business, err := getBusiness(r)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Services initialization error: "+err.Error())
		return
	}
	params := r.URL.Query()
	namespaces := params.Get("namespaces") // csl of namespaces
	ns := strings.Split(namespaces, ",")
	experiments, err := business.Iter8.GetIter8Experiments(ns)
	if err != nil {
		handleErrorResponse(w, err)
		return
	}
	RespondWithJSON(w, http.StatusOK, experiments)
}

func Iter8ExperimentGet(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	business, err := getBusiness(r)
	namespace := params["namespace"]
	name := params["name"]
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Services initialization error: "+err.Error())
		return
	}
	experiment, err := business.Iter8.GetIter8Experiment(namespace, name)
	if err != nil {
		handleErrorResponse(w, err)
		return
	}
	RespondWithJSON(w, http.StatusOK, experiment)
}

func Iter8ExperimentCreate(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	business, err := getBusiness(r)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Services initialization error: "+err.Error())
		return
	}
	namespace := params["namespace"]
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	experiment, err := business.Iter8.CreateIter8Experiment(namespace, body)
	if err != nil {
		handleErrorResponse(w, err)
		return
	}
	RespondWithJSON(w, http.StatusOK, experiment)
}

func Iter8ExperimentUpdate(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	business, err := getBusiness(r)
	namespace := params["namespace"]
	name := params["name"]
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Services initialization error: "+err.Error())
		return
	}
	// TODO This is not yet implemented
	experiment, err := business.Iter8.GetIter8Experiment(namespace, name)
	if err != nil {
		handleErrorResponse(w, err)
		return
	}
	RespondWithJSON(w, http.StatusOK, experiment)
}

func Iter8ExperimentDelete(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	business, err := getBusiness(r)
	namespace := params["namespace"]
	name := params["name"]
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Services initialization error: "+err.Error())
		return
	}
	err = business.Iter8.DeleteIter8Experiment(namespace, name)
	if err != nil {
		handleErrorResponse(w, err)
		return
	}
	RespondWithCode(w, http.StatusOK)
}


func Iter8Dashboard(w http.ResponseWriter, r *http.Request) {
	business, err := getBusiness(r)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Services initialization error: "+err.Error())
		return
	}

	vars := mux.Vars(r)
	namespace := vars["namespace"]
	service := vars["service"]

	prom, namespaceInfo := initClientsForMetrics(w, r, defaultPromClientSupplier, namespace)
	if prom == nil {
		// any returned value nil means error & response already written
		return
	}

	params := prometheus.IstioMetricsQuery{Namespace: namespace, Service: service}
	err = extractIstioMetricsQueryParams(r, &params, namespaceInfo)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	dashboard, err := business.Iter8.GetIter8Dashboard(  r.URL.Query(), vars, namespace, service, prom, params)
	if err != nil {
		handleErrorResponse(w, err)
		return
	}
	RespondWithJSON(w, http.StatusOK, dashboard)
}
