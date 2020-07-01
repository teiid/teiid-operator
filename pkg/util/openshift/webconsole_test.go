package openshift

import (
	"context"
	"fmt"
	"testing"

	"github.com/RHsyseng/operator-utils/pkg/test"
	"k8s.io/apimachinery/pkg/runtime"

	consolev1 "github.com/openshift/api/console/v1"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestCreateConsoleLink(t *testing.T) {
	bcRoute, service, vdb := getConsoleLinkParameters()

	CreateConsoleLink(context.TODO(), bcRoute, service.Client, vdb)
	consoleLinkName := fmt.Sprintf("%s-%s", vdb.ObjectMeta.Name, vdb.Namespace)
	consoleLink := &consolev1.ConsoleLink{}
	err := service.Get(context.TODO(), types.NamespacedName{Name: consoleLinkName}, consoleLink)

	assert.Nil(t, err)
	assert.Equal(t, "https://"+bcRoute.Spec.Host, consoleLink.Spec.Href, "The two routes should be the same.")
	assert.Equal(t, consoleLinkName, consoleLink.ObjectMeta.Name, "ConsoleLink names should be the same.")
	assert.Equal(t, "VirtualDatabase - "+bcRoute.Name, consoleLink.Spec.Text, "Route names should be the same.")

	bcRoute.Spec.Host = "www.sampleURL.com"
	bcRoute.Name = "SecondTestName"

	CreateConsoleLink(context.TODO(), bcRoute, service.Client, vdb)
	consoleLinkName = fmt.Sprintf("%s-%s", vdb.ObjectMeta.Name, vdb.Namespace)
	consoleLink2 := &consolev1.ConsoleLink{}
	err = service.Get(context.TODO(), types.NamespacedName{Name: consoleLinkName}, consoleLink2)

	assert.Equal(t, "https://"+bcRoute.Spec.Host, consoleLink2.Spec.Href)
	assert.Equal(t, "VirtualDatabase - "+bcRoute.Name, consoleLink2.Spec.Text)

	// We modify vdb to change consoleLinkName, which creates a new consoleLink
	vdb.ObjectMeta.Name = "new-test"
	vdb.ObjectMeta.Namespace = "new-vdb-ns"

	CreateConsoleLink(context.TODO(), bcRoute, service.Client, vdb)
	consoleLinkName = fmt.Sprintf("%s-%s", vdb.ObjectMeta.Name, vdb.Namespace)
	consoleLink3 := &consolev1.ConsoleLink{}
	err = service.Get(context.TODO(), types.NamespacedName{Name: consoleLinkName}, consoleLink3)

	assert.Equal(t, vdb.ObjectMeta.Name, consoleLink3.ObjectMeta.Labels["teiid.io/name"])
	assert.Equal(t, vdb.ObjectMeta.Namespace, consoleLink3.Spec.NamespaceDashboard.Namespaces[0])
	assert.NotEqual(t, consoleLink2.ObjectMeta.Name, consoleLink3.ObjectMeta.Name)
	assert.Equal(t, consoleLink2.Spec.Href, consoleLink3.Spec.Href)
	assert.Equal(t, consoleLink2.Spec.Text, consoleLink3.Spec.Text)
}

func TestRemoveConsoleLink(t *testing.T) {
	bcRoute, service, vdb := getConsoleLinkParameters()

	RemoveConsoleLink(context.TODO(), service.Client, vdb)
	consoleLinkName := fmt.Sprintf("%s-%s", vdb.ObjectMeta.Name, vdb.Namespace)
	consoleLink := &consolev1.ConsoleLink{}
	err := service.Get(context.TODO(), types.NamespacedName{Name: consoleLinkName}, consoleLink)

	assert.NotNil(t, err) // We expect err != nil because service.Get could not find consoleLink (never created) (error)

	CreateConsoleLink(context.TODO(), bcRoute, service.Client, vdb)
	consoleLinkName = fmt.Sprintf("%s-%s", vdb.ObjectMeta.Name, vdb.Namespace)
	consoleLink = &consolev1.ConsoleLink{}
	err = service.Get(context.TODO(), types.NamespacedName{Name: consoleLinkName}, consoleLink)

	assert.Nil(t, err) // We expect err == nil because service.Get found consoleLink (no error)

	RemoveConsoleLink(context.TODO(), service.Client, vdb)
	consoleLinkName = fmt.Sprintf("%s-%s", vdb.ObjectMeta.Name, vdb.Namespace)
	consoleLink = &consolev1.ConsoleLink{}
	err = service.Get(context.TODO(), types.NamespacedName{Name: consoleLinkName}, consoleLink)

	assert.NotNil(t, err) // We expect err != nil because service.Get could not find (deleted) consoleLink (error)
}

func getInstance(nsName types.NamespacedName) *v1alpha1.VirtualDatabase {
	return &v1alpha1.VirtualDatabase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsName.Name,
			Namespace: nsName.Namespace,
		},
	}
}

func getConsoleLinkParameters() (route *routev1.Route, c *test.MockPlatformService, virdb *v1alpha1.VirtualDatabase) {
	crNamespace := types.NamespacedName{
		Name:      "test",
		Namespace: "vdb-ns",
	}
	vdb := getInstance(crNamespace)

	var localSchemeBuilder = runtime.SchemeBuilder{}
	st := test.NewMockPlatformServiceBuilder(localSchemeBuilder)

	apiObjects := []runtime.Object{&v1alpha1.VirtualDatabase{}, &v1alpha1.VirtualDatabaseList{}}
	st.WithScheme(apiObjects...)
	service := st.Build()

	bcRoute := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "TestName",
			Namespace: "teiid-ns",
		},
		Spec: routev1.RouteSpec{
			Host: "www.example.com",
		},
	}
	return bcRoute, service, vdb
}
