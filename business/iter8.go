package business

import (
	"encoding/json"
	"math"
	"net/url"
	"time"

	"github.com/kiali/kiali/config"
	"github.com/kiali/kiali/kubernetes"
	"github.com/kiali/kiali/log"
	"github.com/kiali/kiali/models"
	"github.com/kiali/kiali/prometheus"
	"github.com/kiali/kiali/prometheus/internalmetrics"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"


	kbus "github.com/kiali/k-charted/business"
	khttp "github.com/kiali/k-charted/http"
	kchartmodel "github.com/kiali/k-charted/model"
	kchart "github.com/kiali/k-charted/business"
)

type Iter8Service struct {
	k8s           kubernetes.IstioClientInterface
	businessLayer *Layer
}

func (in *Iter8Service) GetIter8Info() models.Iter8Info {
	var err error
	promtimer := internalmetrics.GetGoFunctionMetric("business", "Iter8Service", "GetIter8Info")
	defer promtimer.ObserveNow(&err)

	conf := config.Get()

	// It will be considered enabled if the extension is present in the Kiali configuration and the CRD is enabled on the cluster
	if conf.Extensions.Iter8.Enabled && in.k8s.IsIter8Api() {
		return models.Iter8Info{
			Enabled: true,
		}
	}
	return models.Iter8Info{
		Enabled: false,
	}
}

func (in *Iter8Service) GetIter8Experiment(namespace string, name string) (models.Iter8ExperimentDetail, error) {
	var err error
	promtimer := internalmetrics.GetGoFunctionMetric("business", "Iter8Service", "GetIter8Experiment")
	defer promtimer.ObserveNow(&err)

	iter8ExperimentDetail := models.Iter8ExperimentDetail{}

	iter8ExperimentObject, err := in.k8s.GetIter8Experiment(namespace, name)
	if err != nil {
		return iter8ExperimentDetail, err
	}

	iter8ExperimentDetail.Parse(iter8ExperimentObject)
	return iter8ExperimentDetail, nil
}

func (in *Iter8Service) GetIter8ExperimentsByNamespace(namespace string) ([]models.Iter8ExperimentItem, error) {
	var err error
	promtimer := internalmetrics.GetGoFunctionMetric("business", "Iter8Service", "GetIter8ExperimentsByNamespace")
	defer promtimer.ObserveNow(&err)

	return in.fetchIter8Experiments(namespace)
}

func (in *Iter8Service) GetIter8Experiments(namespaces []string) ([]models.Iter8ExperimentItem, error) {
	var err error
	promtimer := internalmetrics.GetGoFunctionMetric("business", "Iter8Service", "GetIter8Experiments")
	defer promtimer.ObserveNow(&err)

	experiments := make([]models.Iter8ExperimentItem, 0)
	if len(namespaces) == 0 {
		allNamespaces, _ := in.businessLayer.Namespace.GetNamespaces()
		for _, namespace := range allNamespaces {
			namespaces = append(namespaces, namespace.Name)
		}
	}
	for _, namespace := range namespaces {
		experimentsOfNamespace, err := in.fetchIter8Experiments(namespace)
		if err == nil {
			for _, item := range experimentsOfNamespace {
				experiments = append(experiments, item)
			}
		}
	}
	return experiments, nil
}

func (in *Iter8Service) fetchIter8Experiments(namespace string) ([]models.Iter8ExperimentItem, error) {
	iter8ExperimentObjects, err := in.k8s.GetIter8Experiments(namespace)
	if err != nil {
		return []models.Iter8ExperimentItem{}, err
	}
	experiments := make([]models.Iter8ExperimentItem, 0)
	for _, iter8ExperimentObject := range iter8ExperimentObjects {
		iter8ExperimentItem := models.Iter8ExperimentItem{}
		iter8ExperimentItem.Parse(iter8ExperimentObject)
		experiments = append(experiments, iter8ExperimentItem)
	}
	return experiments, nil
}

func (in *Iter8Service) CreateIter8Experiment(namespace string, body []byte) (models.Iter8ExperimentDetail, error) {
	var err error
	promtimer := internalmetrics.GetGoFunctionMetric("business", "Iter8Service", "CreateIter8Experiment")
	defer promtimer.ObserveNow(&err)

	iter8ExperimentDetail := models.Iter8ExperimentDetail{}

	json, err := in.ParseJsonForCreate(body)
	if err != nil {
		return iter8ExperimentDetail, err
	}

	iter8ExperimentObject, err := in.k8s.CreateIter8Experiment(namespace, json)
	if err != nil {
		return iter8ExperimentDetail, err
	}

	iter8ExperimentDetail.Parse(iter8ExperimentObject)
	return iter8ExperimentDetail, nil
}

func (in *Iter8Service) ParseJsonForCreate(body []byte) (string, error) {

	newExperimentSpec := models.Iter8ExperimentSpec{}
	err := json.Unmarshal(body, &newExperimentSpec)
	if err != nil {
		return "", err
	}
	object := kubernetes.Iter8ExperimentObject{
		TypeMeta: v1.TypeMeta{
			APIVersion: kubernetes.Iter8GroupVersion.String(),
			Kind:       "Experiment",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      newExperimentSpec.Name,
			Namespace: newExperimentSpec.Namespace,
		},
		Spec:    kubernetes.Iter8ExperimentSpec{},
		Metrics: nil,
		Status:  kubernetes.Iter8ExperimentStatus{},
	}
	object.Spec.TargetService.ApiVersion = "v1"
	object.Spec.TargetService.Name = newExperimentSpec.Service
	object.Spec.TargetService.Baseline = newExperimentSpec.Baseline
	object.Spec.TargetService.Candidate = newExperimentSpec.Candidate
	object.Spec.TrafficControl.Strategy = newExperimentSpec.TrafficControl.Algorithm
	object.Spec.TrafficControl.MaxTrafficPercentage = newExperimentSpec.TrafficControl.MaxTrafficPercentage

	b, err2 := json.Marshal(object)
	if err2 != nil {
		return "", err2
	}
	return string(b), nil
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}


// NewDashboardsService initializes this business service
func (in *Iter8Service) GetIter8Dashboard( u url.Values, v map[string]string , namespace string,  service string, prom *prometheus.Client, params prometheus.IstioMetricsQuery) (*kchartmodel.MonitoringDashboard, error) {
	log.Infof("Calling Iter8Dashboard ", namespace, " and ", service)
	var iter8Charts []models.Iter8Chart
	handler, chartinfo,  err := Iter8DashboardHandler(u, v)
	for _, ch := range chartinfo.Charts {
		it := models.Iter8Chart {
			Chart: ch,
			RefName: "request_count",
			Scale: 0.0,
		}
		iter8Charts = append(iter8Charts, it)
	}

	dashboard, err :=  GetDashboardMetrics(handler, prom,  params, iter8Charts)
	if err != nil {
		return nil, nil
	}
	return dashboard, nil
}

func Iter8DashboardHandler(queryParams url.Values, pathParams map[string]string)  (*kchart.DashboardsService, *kchartmodel.MonitoringDashboard, error) {
	_cfg, _log := DashboardsConfig()
	namespace := pathParams["namespace"]

	dashboardName := "iter8-metrics"

	svc := kchart.NewDashboardsService(_cfg, _log)

	params := kchartmodel.DashboardQuery{Namespace: namespace}
	err := khttp.ExtractDashboardQueryParams(queryParams, &params)
	log.Infof("ExtractDashboardQueryParams return ", err)
	if err != nil {
		return nil, nil, err
	}
	log.Infof("Getting dashboard ", dashboardName)
	md, err := svc.GetDashboard(params, dashboardName)
	log.Infof("ExtractDashboardQueryParams return ", md)
	return &svc, md, err
}

func GetDashboardMetrics(in *kbus.DashboardsService, prom *prometheus.Client, params prometheus.IstioMetricsQuery, iter8Charts []models.Iter8Chart) (*kchartmodel.MonitoringDashboard, error) {

	var dashboard kchartmodel.MonitoringDashboard
	// Copy dashboard
	if params.Direction == "inbound" {
		dashboard = models.PrepareIstioDashboard("Inbound", "destination", "source")
	} else {
		dashboard = models.PrepareIstioDashboard("Outbound", "source", "destination")
	}

	metrics := prom.GetMetrics(&params)
    log.Infof("After GetMetrics %v", metrics.Metrics)
	// TODO: remove this hacky code when Istio finishes migrating to the millis duration metric,
	//       until then use the one that has data, preferring millis in the corner case that
	//       both have data for the time range.
	_, secondsOK := metrics.Histograms["request_duration"]
	durationMillis, millisOK := metrics.Histograms["request_duration_millis"]
	if secondsOK && millisOK {
		durationMillisEmpty := true
	MillisEmpty:
		for _, samples := range durationMillis {
			for _, sample := range samples.Matrix {
				for _, pair := range sample.Values {
					if !math.IsNaN(float64(pair.Value)) {
						durationMillisEmpty = false
						break MillisEmpty
					}
				}
			}
		}
		if !durationMillisEmpty {
			delete(metrics.Histograms, "request_duration")
		} else {
			delete(metrics.Histograms, "request_duration_millis")
		}
	}

	for _, chartTpl := range iter8Charts {
		newChart := chartTpl.Chart
		unitScale := 1.0
		if chartTpl.Scale != 0.0 {
			unitScale = chartTpl.Scale
		}
		log.Infof("Working on chart refName %s UnitSclae %d",
			chartTpl.RefName , unitScale)

		if metric, ok := metrics.Metrics[chartTpl.RefName]; ok {
			log.Infof("ConvertMatrix %v %d", metric.Matrix, unitScale)

			newChart.Metric = kchartmodel.ConvertMatrix(metric.Matrix, unitScale)
		}
		if histo, ok := metrics.Histograms[chartTpl.RefName]; ok {
			newChart.Histogram = make(map[string][]*kchartmodel.SampleStream, len(histo))
			for k, v := range histo {
				newChart.Histogram[k] = kchartmodel.ConvertMatrix(v.Matrix, unitScale)
			}
		}
		if newChart.Metric != nil || newChart.Histogram != nil {
			dashboard.Charts = append(dashboard.Charts, newChart)
		}
	}
	return &dashboard, nil
}
