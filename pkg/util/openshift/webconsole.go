package openshift

//go:generate go run ./.packr/packr.go

import (
	"context"
	"fmt"

	"github.com/RHsyseng/operator-utils/pkg/logs"
	"github.com/RHsyseng/operator-utils/pkg/utils/kubernetes"
	"github.com/RHsyseng/operator-utils/pkg/utils/openshift"
	"github.com/ghodss/yaml"
	"github.com/gobuffalo/packr/v2"
	consolev1 "github.com/openshift/api/console/v1"
	routev1 "github.com/openshift/api/route/v1"
	v1alpha1 "github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = logs.GetLogger("openshift-webconsole")

// ConsoleYAMLSampleExists --
func ConsoleYAMLSampleExists() error {
	gvk := schema.GroupVersionKind{Group: "console.openshift.io", Version: "v1", Kind: "ConsoleYAMLSample"}
	return kubernetes.CustomResourceDefinitionExists(gvk)
}

// CreateConsoleYAMLSamples --
func CreateConsoleYAMLSamples(c client.Client) {
	log.Info("Loading CR YAML samples.")
	box := packr.New("cryamlsamples", "../../../deploy/crs")
	if box.List() == nil {
		log.Error(nil, "CR YAML folder is empty. It is not loaded.")
		return
	}

	resMap := make(map[string]string)
	for _, filename := range box.List() {
		yamlStr, err := box.FindString(filename)
		if err != nil {
			resMap[filename] = err.Error()
			continue
		}
		teiid := v1alpha1.VirtualDatabase{}
		err = yaml.Unmarshal([]byte(yamlStr), &teiid)
		if err != nil {
			resMap[filename] = err.Error()
			continue
		}
		yamlSample, err := openshift.GetConsoleYAMLSample(&teiid)
		if err != nil {
			resMap[filename] = err.Error()
			continue
		}
		err = c.Create(context.TODO(), yamlSample)
		if err != nil {
			resMap[filename] = err.Error()
			continue
		}
		resMap[filename] = "Applied"
	}

	for k, v := range resMap {
		log.Info("yaml ", "name: ", k, " status: ", v)
	}
}

// ConsoleLinkExists --
func ConsoleLinkExists() error {
	gvk := schema.GroupVersionKind{Group: "console.openshift.io", Version: "v1", Kind: "ConsoleLink"}
	return kubernetes.CustomResourceDefinitionExists(gvk)
}

// CreateConsoleLink --
func CreateConsoleLink(ctx context.Context, route *routev1.Route, c client.Client, vdb *v1alpha1.VirtualDatabase) {
	consoleLinkName := fmt.Sprintf("%s-%s", vdb.ObjectMeta.Name, vdb.Namespace)
	doCreateConsoleLink(ctx, consoleLinkName, route, c, vdb)
}

func doCreateConsoleLink(ctx context.Context, consoleLinkName string, route *routev1.Route, c client.Client, vdb *v1alpha1.VirtualDatabase) {
	if route != nil {
		checkConsoleLink(ctx, route, consoleLinkName, vdb, c)
	}
}

func checkConsoleLink(ctx context.Context, route *routev1.Route, consoleLinkName string, vdb *v1alpha1.VirtualDatabase, c client.Client) {
	consoleLink := &consolev1.ConsoleLink{}
	err := c.Get(ctx, types.NamespacedName{Name: consoleLinkName}, consoleLink)
	if err != nil && apierrors.IsNotFound(err) {
		consoleLink = createNamespaceDashboardLink(consoleLinkName, route, vdb)
		if err := c.Create(ctx, consoleLink); err != nil {
			log.Error(err, "Console link is not created.")
		} else {
			log.Info("Console link has been created. ", consoleLinkName)
		}
	} else if err == nil && consoleLink != nil { // if consoleLink already exists, update consoleLink
		reconcileConsoleLink(ctx, route, consoleLink, c)
	}
}

func reconcileConsoleLink(ctx context.Context, route *routev1.Route, link *consolev1.ConsoleLink, client client.Client) {
	url := "https://" + route.Spec.Host
	linkTxt := consoleLinkText(route)
	if url != link.Spec.Href || linkTxt != link.Spec.Text {
		link.Spec.Href = url
		link.Spec.Text = linkTxt
		if err := client.Update(ctx, link); err != nil {
			log.Error(err, "failed to reconcile Console Link", link)
		}
	}
}

func createNamespaceDashboardLink(consoleLinkName string, route *routev1.Route, vdb *v1alpha1.VirtualDatabase) *consolev1.ConsoleLink {
	return &consolev1.ConsoleLink{
		ObjectMeta: metav1.ObjectMeta{
			Name: consoleLinkName,
			Labels: map[string]string{
				"teiid.io/name": vdb.ObjectMeta.Name,
			},
		},
		Spec: consolev1.ConsoleLinkSpec{
			Link: consolev1.Link{
				Text: consoleLinkText(route),
				Href: "https://" + route.Spec.Host,
			},
			Location: consolev1.NamespaceDashboard,
			NamespaceDashboard: &consolev1.NamespaceDashboardSpec{
				Namespaces: []string{vdb.Namespace},
			},
		},
	}
}

func consoleLinkText(route *routev1.Route) string {
	return "VirtualDatabase - " + route.Name
}

// RemoveConsoleLink --
func RemoveConsoleLink(ctx context.Context, c client.Client, vdb *v1alpha1.VirtualDatabase) {
	consoleLinkName := fmt.Sprintf("%s-%s", vdb.ObjectMeta.Name, vdb.Namespace)
	doDeleteConsoleLink(ctx, consoleLinkName, c)
}

func doDeleteConsoleLink(ctx context.Context, consoleLinkName string, c client.Client) {
	consoleLink := &consolev1.ConsoleLink{}
	err := c.Get(ctx, types.NamespacedName{Name: consoleLinkName}, consoleLink)
	if err == nil && consoleLink != nil {
		err = c.Delete(ctx, consoleLink)
		if err != nil {
			log.Error(err, "Failed to delete the consolelink:", consoleLinkName)
		} else {
			log.Info("deleted the consolelink:", consoleLinkName)
		}
	}
}
