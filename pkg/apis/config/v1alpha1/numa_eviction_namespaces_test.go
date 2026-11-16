/*
Copyright 2022 The Katalyst Authors.

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

package v1alpha1

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

func TestNumaEvictionAllowedNamespacesJSONAndDeepCopy(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		raw     string
		present bool
		want    string
	}{
		{`{}`, false, ""},
		{`{"numaEvictionAllowedNamespaces":null}`, false, ""},
		{`{"numaEvictionAllowedNamespaces":""}`, true, ""},
		{`{"numaEvictionAllowedNamespaces":"a,b"}`, true, "a,b"},
	} {
		tt := tt
		t.Run(tt.raw, func(t *testing.T) {
			t.Parallel()
			var conf MemoryPressureEvictionConfig
			require.NoError(t, json.Unmarshal([]byte(tt.raw), &conf))
			copied := conf.DeepCopy()
			encoded, err := json.Marshal(copied)
			require.NoError(t, err)
			if !tt.present {
				require.Nil(t, copied.NumaEvictionAllowedNamespaces)
				require.NotContains(t, string(encoded), "numaEvictionAllowedNamespaces")
				return
			}
			require.NotNil(t, copied.NumaEvictionAllowedNamespaces)
			require.Equal(t, tt.want, *copied.NumaEvictionAllowedNamespaces)
			require.Contains(t, string(encoded), `"numaEvictionAllowedNamespaces":`)
			*copied.NumaEvictionAllowedNamespaces = "changed"
			require.Equal(t, tt.want, *conf.NumaEvictionAllowedNamespaces)
		})
	}
}

func TestNumaEvictionAllowedNamespacesCRDSchema(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("../../../../config/crd/bases/config.katalyst.kubewharf.io_adminqosconfigurations.yaml")
	require.NoError(t, err)
	var crd map[string]interface{}
	require.NoError(t, yaml.Unmarshal(data, &crd))
	versions := crd["spec"].(map[string]interface{})["versions"].([]interface{})
	for _, version := range versions {
		schema := version.(map[string]interface{})["schema"].(map[string]interface{})["openAPIV3Schema"].(map[string]interface{})
		for _, field := range []string{"spec", "config", "evictionConfig", "memoryPressureEvictionConfig", "numaEvictionAllowedNamespaces"} {
			schema = schema["properties"].(map[string]interface{})[field].(map[string]interface{})
		}
		require.Equal(t, "string", schema["type"])
		pattern, err := regexp.Compile(schema["pattern"].(string))
		require.NoError(t, err)
		for _, value := range []string{"", "default", "a,b", "ns-1", "a,a", strings.Repeat("a", 63)} {
			require.True(t, pattern.MatchString(value), "must allow %q", value)
		}
		for _, value := range []string{" ", ",", "a,", ",a", "a,,b", "a, b", "a;b", "A", "a.b", "-a", "a-", strings.Repeat("a", 64)} {
			require.False(t, pattern.MatchString(value), "must reject %q", value)
		}
	}
}
