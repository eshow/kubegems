// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package prometheus

import (
	"testing"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	alertmanagertypes "github.com/prometheus/alertmanager/types"

	"github.com/google/go-cmp/cmp"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestRawAlertResource_ToAlerts(t *testing.T) {
	type fields struct {
		AlertmanagerConfig *v1alpha1.AlertmanagerConfig
		PrometheusRule     *monitoringv1.PrometheusRule
		Silences           []alertmanagertypes.Silence
	}
	type args struct {
		containOrigin bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    AlertRuleList[MonitorAlertRule]
		wantErr bool
	}{
		{
			name: "1",
			fields: fields{
				AlertmanagerConfig: &v1alpha1.AlertmanagerConfig{
					ObjectMeta: v1.ObjectMeta{
						Name:      "myconfig",
						Namespace: GlobalAlertNamespace,
					},
					Spec: v1alpha1.AlertmanagerConfigSpec{
						Receivers: []v1alpha1.Receiver{
							NullReceiver,
						},
						Route: &v1alpha1.Route{
							Receiver: NullReceiverName,
							Routes: []extv1.JSON{
								{
									Raw: []byte(`
									{
										"receiver": "receiver-1",
										"matchers": [
											{
												"name": "gems_alertname",
												"value": "alert-1"
											},
											{
												"name": "gems_namespace",
												"value": "kubegems-monitoring"
											}
										]
									}`),
								},
							},
						},
					},
				},
				PrometheusRule: &monitoringv1.PrometheusRule{
					ObjectMeta: v1.ObjectMeta{
						Name:      "myrule",
						Namespace: GlobalAlertNamespace,
					},
					Spec: monitoringv1.PrometheusRuleSpec{
						Groups: []monitoringv1.RuleGroup{
							{
								Name: "alert-1",
								Rules: []monitoringv1.Rule{
									{
										Alert: "alert-1",
										Expr:  intstr.FromString(`kube_node_status_condition{condition=~"Ready", status=~"true"}==0`),
										For:   "1m",
										Labels: map[string]string{
											AlertNameLabel:      "alert-1",
											AlertNamespaceLabel: GlobalAlertNamespace,
											SeverityLabel:       SeverityError,
										},
										Annotations: map[string]string{
											ExprJsonAnnotationKey: `{
												"scope": "system",
												"resource": "node",
												"rule": "statusCondition",
												"unit": "",
												"labelpairs": {
													"condition": "Ready",
													"status": "true"
												}
											}`,
										},
									},
								},
							},
						},
					},
				},
				Silences: nil,
			},
			args: args{
				containOrigin: false,
			},
			want: AlertRuleList[MonitorAlertRule]{{
				BaseAlertRule: BaseAlertRule{
					Namespace: GlobalAlertNamespace,
					Name:      "alert-1",
					Expr:      `kube_node_status_condition{condition=~"Ready", status=~"true"}`,
					For:       "1m",
					AlertLevels: []AlertLevel{{
						CompareOp:    "==",
						CompareValue: "0",
						Severity:     SeverityError,
					}},
					Receivers:     []AlertReceiver{{Name: "receiver-1"}},
					IsOpen:        true,
					InhibitLabels: []string{},
				},

				PromqlGenerator: &PromqlGenerator{
					Scope:    "system",
					Resource: "node",
					Rule:     "statusCondition",
					LabelPairs: map[string]string{
						"condition": "Ready",
						"status":    "true",
					},
				},
				Source: "myrule",
			}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := &RawMonitorAlertResource{
				Base: &BaseAlertResource{
					AMConfig: tt.fields.AlertmanagerConfig,
					Silences: tt.fields.Silences,
				},
				PrometheusRule: tt.fields.PrometheusRule,
			}
			got, err := raw.ToAlerts(tt.args.containOrigin)
			if (err != nil) != tt.wantErr {
				t.Errorf("RawAlertResource.ToAlerts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("RawAlertResource.ToAlerts() = %v, want %v", got, tt.want)
				t.Error("diff: ", diff)
			}
		})
	}
}
