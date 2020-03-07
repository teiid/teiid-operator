package test

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RHsyseng/operator-utils/pkg/validation"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
)

func TestSampleCustomResources(t *testing.T) {
	schema := getSchema(t)
	fileList := fileList("../../../../deploy/crs")
	for _, filePath := range fileList {
		file, err := os.Open(filePath)
		if err != nil {
			log.Println(err)
		}
		defer file.Close()
		yamlString, err := ioutil.ReadAll(file)
		assert.NoError(t, err, "Error reading %v CR yaml", filePath)
		var input map[string]interface{}
		assert.NoError(t, yaml.Unmarshal(yamlString, &input))
		assert.NoError(t, schema.Validate(input), "File %v does not validate against the CRD schema", filePath)
	}
}

func TestTrialEnvMinimum(t *testing.T) {
	var inputYaml = `
apiVersion: teiid.io/v1alpha1
kind: VirtualDatabase
metadata:
  name: trial
spec:
  build:
    git:
      uri: https://github.com/teiid/teiid-openshift-examples
`
	var input map[string]interface{}
	assert.NoError(t, yaml.Unmarshal([]byte(inputYaml), &input))

	schema := getSchema(t)
	assert.NoError(t, schema.Validate(input))

	//	deleteNestedMapEntry(input, "spec", "environment")
	//	assert.Error(t, schema.Validate(input))
}

func TestCompleteCRD(t *testing.T) {
	schema := getSchema(t)
	missingEntries := schema.GetMissingEntries(&v1alpha1.VirtualDatabase{})
	for _, missing := range missingEntries {
		if strings.HasPrefix(missing.Path, "/status") {
			//Not using subresources, so status is not expected to appear in CRD
		} else if strings.Contains(missing.Path, "/env/valueFrom/") {
			//The valueFrom is not expected to be used and is not fully defined TODO: verify
		} else if strings.Contains(missing.Path, "/spec/datasources/") {
			//The valueFrom is not expected to be used and is not fully defined TODO: verify
		} else if strings.HasSuffix(missing.Path, "/from/uid") {
			//The ObjectReference in From is not expected to be used and is not fully defined TODO: verify
		} else if strings.HasSuffix(missing.Path, "/from/apiVersion") {
			//The ObjectReference in From is not expected to be used and is not fully defined TODO: verify
		} else if strings.HasSuffix(missing.Path, "/from/resourceVersion") {
			//The ObjectReference in From is not expected to be used and is not fully defined TODO: verify
		} else if strings.HasSuffix(missing.Path, "/from/fieldPath") {
			//The ObjectReference in From is not expected to be used and is not fully defined TODO: verify
		} else if strings.HasSuffix(missing.Path, "/spec/exposeVia3scale") {
			//The ObjectReference in From is not expected to be used and is not fully defined TODO: verify
		} else if strings.HasSuffix(missing.Path, "/spec/build/source/openapi") {
			//The ObjectReference in From is not expected to be used and is not fully defined TODO: verify
		} else if strings.HasSuffix(missing.Path, "/spec/build/source/maven") {
			//The ObjectReference in From is not expected to be used and is not fully defined TODO: verify
		} else {
			assert.Fail(t, "Discrepancy between CRD and Struct", "Missing or incorrect schema validation at %v, expected type %v", missing.Path, missing.Type)
		}
	}
}

func deleteNestedMapEntry(object map[string]interface{}, keys ...string) {
	for index := 0; index < len(keys)-1; index++ {
		object = object[keys[index]].(map[string]interface{})
	}
	delete(object, keys[len(keys)-1])
}

func getSchema(t *testing.T) validation.Schema {
	crdFile := "../../../../deploy/crds/teiid.io_virtualdatabases_crd.yaml"
	file, err := os.Open(crdFile)
	if err != nil {
		log.Println(err)
	}
	defer file.Close()
	yamlString, err := ioutil.ReadAll(file)
	assert.NoError(t, err, "Error reading CRD yaml %v", yamlString)
	schema, err := validation.New(yamlString)
	assert.NoError(t, err)
	return schema
}

func fileList(dir string) []string {
	var fileList []string
	err := filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				fileList = append(fileList, path)
			}
			return nil
		})
	if err != nil {
		log.Println(err)
		return fileList
	}
	fmt.Println(fileList)
	return fileList
}
