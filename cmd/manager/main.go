package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	kubemetrics "github.com/operator-framework/operator-sdk/pkg/kube-metrics"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"os"
	"runtime"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/operator-framework/operator-sdk/pkg/metrics"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/spf13/pflag"
	"github.com/teiid/teiid-operator/pkg/apis"
	"github.com/teiid/teiid-operator/pkg/controller"
	"github.com/teiid/teiid-operator/pkg/util/logs"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	//_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost               = "0.0.0.0"
	metricsPort         int32 = 8383
	log                       = logs.GetLogger("cmd")
	operatorMetricsPort int32 = 8686
)

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Version of operator-sdk: %v", sdkVersion.Version))
}

func main() {
	// Add the zap logger flag set to the CLI. The flag set must
	// be added before calling pflag.Parse().
	pflag.CommandLine.AddFlagSet(zap.FlagSet())

	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	pflag.Parse()

	printVersion()

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.Error(err, "Failed to get watch namespace")
		os.Exit(1)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	ctx := context.TODO()

	// Become the leader before proceeding
	err = leader.Become(ctx, "teiid-operator-lock")
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:          namespace,
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
	})
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Add the Metrics Service
	addMetrics(ctx, cfg)

	log.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}

func addMonitoringKeyLabelToOperatorServiceMonitor(serviceMonitor *monitoringv1.ServiceMonitor) error {
	updatedLabels := map[string]string{"monitoring-key": "middleware"}
	for k, v := range serviceMonitor.ObjectMeta.Labels {
		updatedLabels[k] = v
	}
	serviceMonitor.SetLabels(updatedLabels)

	return nil
}

// addMetrics will create the Services and Service Monitors to allow the operator export the metrics by using
// the Prometheus operator
func addMetrics(ctx context.Context, cfg *rest.Config) {
	if err := serveCRMetrics(cfg); err != nil {
		if errors.Is(err, k8sutil.ErrRunLocal) {
			log.Info("Skipping CR metrics server creation; not running in a cluster.")
			return
		}
		log.Info("Could not generate and serve custom resource metrics", "error", err.Error())
	}

	// Add to the below struct any other metrics ports you want to expose.
	servicePorts := []v1.ServicePort{
		{Port: metricsPort, Name: metrics.OperatorPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: metricsPort}},
		{Port: operatorMetricsPort, Name: metrics.CRPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: operatorMetricsPort}},
	}

	// Create Service object to expose the metrics port(s).
	service, err := metrics.CreateMetricsService(ctx, cfg, servicePorts)
	if err != nil {
		log.Info("Could not create metrics Service", "error", err.Error())
	}

	// Retrieve the namespace the operator is running in
	operatorNs, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		log.Error(err, "Failed to get operator namespace")
	}

	// CreateServiceMonitors will automatically create the prometheus-operator ServiceMonitor resources
	// necessary to configure Prometheus to scrape metrics from this operator.
	services := []*v1.Service{service}
	_, err = metrics.CreateServiceMonitors(cfg, operatorNs, services, addMonitoringKeyLabelToOperatorServiceMonitor)
	if err != nil {
		log.Info("Could not create ServiceMonitor object", "error", err.Error())
		// If this operator is deployed to a cluster without the prometheus-operator running, it will return
		// ErrServiceMonitorNotPresent, which can be used to safely skip ServiceMonitor creation.
		if err == metrics.ErrServiceMonitorNotPresent {
			log.Info("Install prometheus-operator in your cluster to create ServiceMonitor objects", "error", err.Error())
		}
	}
}

// serveCRMetrics gets the Operator/CustomResource GVKs and generates metrics based on those types.
// It serves those metrics on "http://metricsHost:operatorMetricsPort".
func serveCRMetrics(cfg *rest.Config) error {
	// Below function returns filtered operator/CustomResource specific GVKs.
	// For more control override the below GVK list with your own custom logic.
	gvks, err := k8sutil.GetGVKsFromAddToScheme(apis.AddToScheme)
	if err != nil {
		return err
	}

	// We perform our custom GKV filtering on top of the one performed
	// by operator-sdk code
	filteredGVK := filterGKVsFromAddToScheme(gvks)
	if err != nil {
		return err
	}
	// Get the namespace the operator is currently deployed in.
	operatorNs, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		return err
	}
	// To generate metrics in other namespaces, add the values below.
	ns := []string{operatorNs}
	// Generate and serve custom resource specific metrics.
	err = kubemetrics.GenerateAndServeCRMetrics(cfg, ns, filteredGVK, metricsHost, operatorMetricsPort)
	if err != nil {
		return err
	}
	return nil
}

func filterGKVsFromAddToScheme(gvks []schema.GroupVersionKind) []schema.GroupVersionKind {
	// We use gkvFilters to filter from the existing GKVs defined in the used
	// runtime.Schema for the operator. The reason for that is that
	// kube-metrics tries to list all of the defined Kinds in the schemas
	// that are passed, including Kinds that the operator doesn't use and
	// thus the role used the operator doesn't have them set and we don't want
	// to set as they are not used by the operator.
	// For the fields that the filters have we have defined the value '*' to
	// specify any will be a match (accepted)
	matchAnyValue := "*"
	gvkFilters := []schema.GroupVersionKind{
		// OpenShift types
		schema.GroupVersionKind{Group: "route.openshift.io", Kind: "Route", Version: matchAnyValue},
		schema.GroupVersionKind{Group: "image.openshift.io", Kind: "ImageStream", Version: matchAnyValue},
		schema.GroupVersionKind{Group: "apps.openshift.io", Kind: "DeploymentConfig", Version: matchAnyValue},
	}

	ownGVKs := []schema.GroupVersionKind{}
	for _, gvk := range gvks {
		for _, gvkFilter := range gvkFilters {
			match := true
			if gvkFilter.Kind == matchAnyValue && gvkFilter.Group == matchAnyValue && gvkFilter.Version == matchAnyValue {
				log.Info("gvkFilter should at least have one of its fields defined. Skipping...")
				match = false
			} else {
				if gvkFilter.Kind != matchAnyValue && gvkFilter.Kind != gvk.Kind {
					match = false
				}
				if gvkFilter.Group != matchAnyValue && gvkFilter.Group != gvk.Group {
					match = false
				}
				if gvkFilter.Version != matchAnyValue && gvkFilter.Version != gvk.Version {
					match = false
				}
			}
			if match {
				ownGVKs = append(ownGVKs, gvk)
			}
		}
	}

	return ownGVKs
}