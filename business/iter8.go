package business

import (
	"encoding/json"
	"math"
	"net/url"
	"strings"
	"sync"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kiali/kiali/config"
	"github.com/kiali/kiali/kubernetes"
	"github.com/kiali/kiali/log"
	"github.com/kiali/kiali/models"
	"github.com/kiali/kiali/prometheus"
	"github.com/kiali/kiali/prometheus/internalmetrics"

	kbus "github.com/kiali/k-charted/business"
	khttp "github.com/kiali/k-charted/http"
	kchartmodel "github.com/kiali/k-charted/model"
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

	errChan := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)

	var iter8ExperimentObject kubernetes.Iter8Experiment
	var canCreate, canUpdate, canDelete bool

	go func(errChan chan error) {
		defer wg.Done()
		var gErr error
		iter8ExperimentObject, gErr = in.k8s.GetIter8Experiment(namespace, name)
		if gErr == nil {
			iter8ExperimentDetail.Parse(iter8ExperimentObject)
		} else {
			errChan <- gErr
		}

	}(errChan)

	go func(errChan chan error) {
		defer wg.Done()
		canCreate, canUpdate, canDelete = getPermissions(in.k8s, namespace, Experiments, "")
	}(errChan)

	wg.Wait()
	if len(errChan) != 0 {
		err = <-errChan
		return iter8ExperimentDetail, err
	}

	iter8ExperimentDetail.Permissions.Create = canCreate
	iter8ExperimentDetail.Permissions.Update = canUpdate
	iter8ExperimentDetail.Permissions.Delete = canDelete

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
			experiments = append(experiments, experimentsOfNamespace...)
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
			Name: newExperimentSpec.Name,
		},
		Spec:    kubernetes.Iter8ExperimentSpec{},
		Metrics: kubernetes.Iter8ExperimentMetrics{},
		Status:  kubernetes.Iter8ExperimentStatus{},
	}
	object.Spec.TargetService.ApiVersion = "v1"
	object.Spec.TargetService.Name = newExperimentSpec.Service
	object.Spec.TargetService.Baseline = newExperimentSpec.Baseline
	object.Spec.TargetService.Candidate = newExperimentSpec.Candidate
	object.Spec.TrafficControl.Strategy = newExperimentSpec.TrafficControl.Algorithm
	object.Spec.TrafficControl.MaxTrafficPercentage = newExperimentSpec.TrafficControl.MaxTrafficPercentage
	object.Spec.TrafficControl.MaxIterations = newExperimentSpec.TrafficControl.MaxIterations
	object.Spec.TrafficControl.TrafficStepSize = newExperimentSpec.TrafficControl.TrafficStepSize

	b, err2 := json.Marshal(object)
	if err2 != nil {
		return "", err2
	}
	return string(b), nil
}

func (in *Iter8Service) DeleteIter8Experiment(namespace string, name string) (err error) {
	promtimer := internalmetrics.GetGoFunctionMetric("business", "Iter8Service", "DeleteIter8Experiment")
	defer promtimer.ObserveNow(&err)

	err = in.k8s.DeleteIter8Experiment(namespace, name)
	return err
}


func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

var iter8SupportCharts = []models.Iter8Chart{
	{
		Chart: kchartmodel.Chart{
			Name:  "Request volume",
			Unit:  "ops",
			Spans: 6,
		},
		RefName: "request_count",
	},
	// TODO: Istio is transitioning from duration in seconds to duration in ms (a new metric). When
	//       complete we should reduce the next two entries to just one entry.
	{
		Chart: kchartmodel.Chart{
			Name:  "Request duration",
			Unit:  "seconds",
			Spans: 6,
		},
		RefName: "request_duration",
	},
	{
		Chart: kchartmodel.Chart{
			Name:  "Request duration",
			Unit:  "seconds",
			Spans: 6,
		},
		RefName: "request_duration_millis",
		Scale:   0.001,
	},
	{
		Chart: kchartmodel.Chart{
			Name:  "Request size",
			Unit:  "bytes",
			Spans: 6,
		},
		RefName: "request_size",
	},
	{
		Chart: kchartmodel.Chart{
			Name:  "Response size",
			Unit:  "bytes",
			Spans: 6,
		},
		RefName: "response_size",
	},
	{
		Chart: kchartmodel.Chart{
			Name:  "TCP received",
			Unit:  "bitrate",
			Spans: 6,
		},
		RefName: "tcp_received",
	},
	{
		Chart: kchartmodel.Chart{
			Name:  "TCP sent",
			Unit:  "bitrate",
			Spans: 6,
		},
		RefName: "tcp_sent",
	},
}

type iter8Metric struct {
	iter8Name      string
	requestName    string
}

var iter8Metrics = []iter8Metric{
	{
		iter8Name: "request_count",
		requestName: "Request Volume",
	},
	{
		iter8Name:      "request_duration",
		requestName:      "Request Duration",
	},
}

func convertChart(monitoringCharts []string) []string {
	var convertedChart []string
	for _, ch := range monitoringCharts {
		for _, n := range iter8Metrics {
			if n.iter8Name == ch {
				convertedChart = append(convertedChart, n.requestName)
			}
		}
	}
	return convertedChart
}
// NewDashboardsService initializes this business service
func (in *Iter8Service) GetIter8Dashboard( u url.Values, v map[string]string , namespace string,  service string, prom *prometheus.Client, params prometheus.IstioMetricsQuery) (*kchartmodel.MonitoringDashboard, error) {
	log.Infof("Calling Iter8Dashboard ", namespace, " and ", service)
	var iter8Charts []models.Iter8Chart
	handler, chartinfo,  err := Iter8DashboardHandler(u, v)
	var monitoringCharts []string

	if charts := u.Get("charts"); charts != "" {
		monitoringCharts = strings.Split(charts, ",")
	} else {
		monitoringCharts = append(monitoringCharts, "request_count")
	}
	iter8NameCharts := convertChart(monitoringCharts)
	log.Infof("MonitoringCharts %v", monitoringCharts)
	log.Infof("Iter8NameCharts %v", iter8NameCharts)
	for _, ch := range chartinfo.Charts {
		log.Infof("Chart Info is %v", ch)
		for i, n := range iter8NameCharts {
			log.Infof("Checking %s and %s", n, ch.Name)
			if n == ch.Name {
				it := models.Iter8Chart {
					Chart: ch,
					RefName: monitoringCharts[i],
					Scale: 0.0,
				}
				iter8Charts = append(iter8Charts, it)
			}
		}
	}

	dashboard, err :=  GetDashboardMetrics(handler, prom,  params, iter8Charts)
	if err != nil {
		return nil, nil
	}

	if (u.Get("baseline") != "" && u.Get("candidate") != "" ) {
		dashboard = filterDashboard(dashboard, u.Get("baseline"), u.Get("candidate"))
	}

	return dashboard, nil
}
func filterDashboard(dashboard *kchartmodel.MonitoringDashboard, baseline string, candidate string ) *kchartmodel.MonitoringDashboard {
	 for _, chart  := range dashboard.Charts {
	 	filterMetric  := make( []*kchartmodel.SampleStream, 2)

		for _, metric := range chart.Metric {
			if (metric.LabelSet["destination_version"] == baseline || metric.LabelSet["destination_version"] == candidate) {
				filterMetric =  append(filterMetric, metric)
			}
		}
		chart.Metric = filterMetric
	}
	return dashboard
}

func Iter8DashboardHandler(queryParams url.Values, pathParams map[string]string)  (*kbus.DashboardsService, *kchartmodel.MonitoringDashboard, error) {
	_cfg, _log := DashboardsConfig()
	namespace := pathParams["namespace"]

	dashboardName := "iter8-metrics"

	svc := kbus.NewDashboardsService(_cfg, _log)

	params := kchartmodel.DashboardQuery{Namespace: namespace}
	err := khttp.ExtractDashboardQueryParams(queryParams, &params)
	log.Infof("ExtractDashboardQueryParams return ", err)
	if err != nil {
		return nil, nil, err
	}
	log.Infof("Getting dashboard ", dashboardName)
	md, err := svc.GetDashboard(params, dashboardName)
	log.Infof("ExtractDashboardQueryParams return %v", md)
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

