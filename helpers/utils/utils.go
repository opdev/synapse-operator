/*
Copyright 2025.

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

package utils

import "gopkg.in/yaml.v3"

func ConvertStructToMap(in any) (map[string]any, error) {
	var intermediate []byte
	var out map[string]any
	intermediate, err := yaml.Marshal(in)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(intermediate, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func BoolToYesNo(report_stats bool) string {
	if report_stats {
		return "yes"
	}
	return "no"
}

func BoolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
