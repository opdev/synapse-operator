//
// This file contains unit tests for the Synapse package
//

package synapse

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	synapsev1alpha1 "github.com/opdev/synapse-operator/api/synapse/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Unit tests for Synapse package", Label("unit"), func() {
	// Testing ParseHomeserverConfigMap
	Context("When parsing the homeserver ConfigMap", func() {
		var r SynapseReconciler
		var ctx context.Context
		var s synapsev1alpha1.Synapse
		var cm corev1.ConfigMap
		var data map[string]string

		BeforeEach(func() {
			// Init variables needed for each test
			r = SynapseReconciler{}
			ctx = context.Background()
			s = synapsev1alpha1.Synapse{}
		})

		Context("Extracting the server name and report stats value from ConfigMap", func() {
			JustBeforeEach(func() {
				cm = corev1.ConfigMap{
					Data: data,
				}
			})

			When("when server name is valid and report stat set to true", func() {
				BeforeEach(func() {
					data = map[string]string{
						"homeserver.yaml": "server_name: my-server-name\nreport_stats: true",
					}
				})

				It("should accordingly update the Synapse Status", func() {
					Expect(r.ParseHomeserverConfigMap(ctx, &s, cm)).Should(Succeed())
					Expect(s.Status.HomeserverConfiguration.ServerName).Should(Equal("my-server-name"))
					Expect(s.Status.HomeserverConfiguration.ReportStats).Should(BeTrue())
				})
			})

			When("when server name is valid and report stat set to false", func() {
				BeforeEach(func() {
					data = map[string]string{
						"homeserver.yaml": "server_name: my-server-name\nreport_stats: false",
					}
				})

				It("should accordingly update the Synapse Status", func() {
					Expect(r.ParseHomeserverConfigMap(ctx, &s, cm)).Should(Succeed())
					Expect(s.Status.HomeserverConfiguration.ServerName).Should(Equal("my-server-name"))
					Expect(s.Status.HomeserverConfiguration.ReportStats).Should(BeFalse())
				})
			})

			When("when 'homeserver.yaml' is not present in the ConfigMap data", func() {
				BeforeEach(func() {
					data = map[string]string{
						"not-homeserver.yaml": "server_name: my-server-name\nreport_stats: true",
					}
				})

				It("should fail to parse the ConfigMap data", func() {
					Expect(r.ParseHomeserverConfigMap(ctx, &s, cm)).ShouldNot(Succeed())
				})
			})

			When("when 'server_name' is not present in the 'homeserver.yaml'", func() {
				BeforeEach(func() {
					data = map[string]string{
						"homeserver.yaml": "not_server_name: my-server-name\nreport_stats: true",
					}
				})

				It("should fail to parse the ConfigMap data", func() {
					Expect(r.ParseHomeserverConfigMap(ctx, &s, cm)).ShouldNot(Succeed())
				})
			})

			When("when 'server_name' is not a valid string", func() {
				BeforeEach(func() {
					data = map[string]string{
						"homeserver.yaml": "server_name: 1234567890\nreport_stats: true",
					}
				})

				It("should fail to parse the ConfigMap data", func() {
					Expect(r.ParseHomeserverConfigMap(ctx, &s, cm)).ShouldNot(Succeed())
				})
			})

			When("when 'report_stats' is not present in the 'homeserver.yaml'", func() {
				BeforeEach(func() {
					data = map[string]string{
						"homeserver.yaml": "server_name: my-server-name\nnot_report_stats: true",
					}
				})

				It("should fail to parse the ConfigMap data", func() {
					Expect(r.ParseHomeserverConfigMap(ctx, &s, cm)).ShouldNot(Succeed())
				})
			})

			When("when 'report_stats' is not a valid bool", func() {
				BeforeEach(func() {
					data = map[string]string{
						"homeserver.yaml": "server_name: my-server-name\nreport_stats: not-a-bool",
					}
				})

				It("should fail to parse the ConfigMap data", func() {
					Expect(r.ParseHomeserverConfigMap(ctx, &s, cm)).ShouldNot(Succeed())
				})
			})

			When("when 'homeserver.yaml' is not valid YAML", func() {
				BeforeEach(func() {
					data = map[string]string{
						"homeserver.yaml": "not valid YAML!",
					}
				})

				It("should fail to parse the ConfigMap data", func() {
					Expect(r.ParseHomeserverConfigMap(ctx, &s, cm)).ShouldNot(Succeed())
				})
			})
		})
	})
})
