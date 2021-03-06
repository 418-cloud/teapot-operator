/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	traefikv1alpha1 "github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefik/v1alpha1"

	azurev1alpha1 "github.com/418-cloud/teapot-operator/apis/azure/v1alpha1"
	configv2 "github.com/418-cloud/teapot-operator/apis/config/v2"
	k8sv1alpha1 "github.com/418-cloud/teapot-operator/apis/k8s/v1alpha1"
	azurecontrollers "github.com/418-cloud/teapot-operator/controllers/azure"
	controllers "github.com/418-cloud/teapot-operator/controllers/k8s"
	azureclient "github.com/418-cloud/teapot-operator/pkg/azure/client"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(k8sv1alpha1.AddToScheme(scheme))
	utilruntime.Must(traefikv1alpha1.AddToScheme(scheme))
	utilruntime.Must(configv2.AddToScheme(scheme))
	utilruntime.Must(azurev1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var configFile string
	var sub string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&configFile, "config", "", "The path to the config file.")
	flag.StringVar(&sub, "subscription", "", "The subscription id to use.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	var err error
	ctrlConfig := configv2.ProjectConfig{
		Subscription: sub,
	}
	options := ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "b1ba06c1.418.cloud",
	}
	if configFile != "" {
		options, err = options.AndFrom(ctrl.ConfigFile().AtPath(configFile).OfKind(&ctrlConfig))
		if err != nil {
			setupLog.Error(err, "unable to load the config file")
			os.Exit(1)
		}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.TeapotAppReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Config: ctrlConfig,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TeapotApp")
		os.Exit(1)
	}
	c, err := readClientConfig("/azure-client-config.json")
	if err != nil {
		setupLog.Error(err, "unable to read the azure client config. Not setting up azure based controllers")
	} else {
		if err = (&azurecontrollers.ContainerEnvironmentReconciler{
			Client:       mgr.GetClient(),
			Scheme:       mgr.GetScheme(),
			Config:       ctrlConfig,
			ClientConfig: c,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "ContainerEnvironment")
			os.Exit(1)
		}
		if err = (&azurecontrollers.ContainerAppReconciler{
			Client:       mgr.GetClient(),
			Scheme:       mgr.GetScheme(),
			Config:       ctrlConfig,
			ClientConfig: c,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "ContainerApp")
			os.Exit(1)
		}
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func readClientConfig(jsonFile string) (c azureclient.ClientConfig, err error) {
	jsonBytes, err := ioutil.ReadFile(jsonFile)
	if err != nil {
		return azureclient.ClientConfig{}, err
	}
	err = json.Unmarshal(jsonBytes, &c)
	return
}
