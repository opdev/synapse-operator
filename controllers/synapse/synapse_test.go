//
//This file contains unit tests for the Synapse package
//

package synapse

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"

	pgov1beta1 "github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
	synapsev1alpha1 "github.com/opdev/synapse-operator/apis/synapse/v1alpha1"
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

	Context("When checking if the PostreSQL Cluster is ready", func() {
		var postgresCluster pgov1beta1.PostgresCluster
		var postgresInstanceSetSpec []pgov1beta1.PostgresInstanceSetSpec
		var postgresInstanceSetStatus []pgov1beta1.PostgresInstanceSetStatus
		var r SynapseReconciler
		var repl int32

		BeforeEach(func() {
			r = SynapseReconciler{}

			// Default PostgresCluster state, to be overwritten in BeforeEach
			repl = int32(1)
			postgresInstanceSetSpec = []pgov1beta1.PostgresInstanceSetSpec{{
				Name:     "instance1",
				Replicas: &repl,
			}}
			postgresInstanceSetStatus = []pgov1beta1.PostgresInstanceSetStatus{{
				Name:            "instance1",
				ReadyReplicas:   1,
				Replicas:        1,
				UpdatedReplicas: 1,
			}}
		})

		JustBeforeEach(func() {
			postgresCluster.Spec.InstanceSets = postgresInstanceSetSpec
			postgresCluster.Status.InstanceSets = postgresInstanceSetStatus
		})

		When("when all replicas are ready and have the desired specification", func() {
			// No BeforeEach, use default PostgresCluster state
			It("Should consider the PostgreSQL cluster as ready", func() {
				Expect(r.isPostgresClusterReady(postgresCluster)).Should(BeTrue())
			})
		})

		When("when some replicas are not ready", func() {
			BeforeEach(func() {
				postgresInstanceSetStatus = []pgov1beta1.PostgresInstanceSetStatus{{
					Name:            "instance1",
					ReadyReplicas:   0,
					Replicas:        1,
					UpdatedReplicas: 1,
				}}
			})

			It("Should not consider the PostgreSQL cluster as ready", func() {
				Expect(r.isPostgresClusterReady(postgresCluster)).Should(BeFalse())
			})
		})

		When("when some replicas are not updated", func() {
			BeforeEach(func() {
				postgresInstanceSetStatus = []pgov1beta1.PostgresInstanceSetStatus{{
					Name:            "instance1",
					ReadyReplicas:   1,
					Replicas:        1,
					UpdatedReplicas: 0,
				}}
			})

			It("Should not consider the PostgreSQL cluster as ready", func() {
				Expect(r.isPostgresClusterReady(postgresCluster)).Should(BeFalse())
			})
		})

		When("when no replicas exist", func() {
			BeforeEach(func() {
				postgresInstanceSetStatus = []pgov1beta1.PostgresInstanceSetStatus{{
					Name:            "instance1",
					ReadyReplicas:   0,
					Replicas:        0,
					UpdatedReplicas: 0,
				}}
			})

			It("Should not consider the PostgreSQL cluster as ready", func() {
				Expect(r.isPostgresClusterReady(postgresCluster)).Should(BeFalse())
			})
		})

		When("when no enough replicas exist", func() {
			BeforeEach(func() {
				repl = int32(2)
				postgresInstanceSetSpec = []pgov1beta1.PostgresInstanceSetSpec{{
					Name:     "instance1",
					Replicas: &repl,
				}}
			})

			It("Should not consider the PostgreSQL cluster as ready", func() {
				Expect(r.isPostgresClusterReady(postgresCluster)).Should(BeFalse())
			})
		})

		When("when an instance is present in spec but not in status", func() {
			BeforeEach(func() {
				repl = int32(1)
				postgresInstanceSetSpec = []pgov1beta1.PostgresInstanceSetSpec{{
					Name:     "instance1",
					Replicas: &repl,
				}, {
					Name:     "instance2",
					Replicas: &repl,
				}}
			})

			It("Should not consider the PostgreSQL cluster as ready", func() {
				Expect(r.isPostgresClusterReady(postgresCluster)).Should(BeFalse())
			})
		})

	})

	Context("When updating the Synapse Status with PostgreSQL database information", func() {
		var r SynapseReconciler
		var s synapsev1alpha1.Synapse
		var synapseDatabaseInfo synapsev1alpha1.SynapseStatusDatabaseConnectionInfo
		// var cm corev1.ConfigMap
		// var homeserver_in map[interface{}]interface{}
		// var homeserver_out map[interface{}]interface{}
		var postgresSecret corev1.Secret
		var postgresSecretData map[string][]byte

		// Re-usable test for checking different happy paths
		check_happy_path := func() {
			By("Updating the Synapse Status")
			Expect(r.updateSynapseStatusDatabase(&s, postgresSecret)).Should(Succeed())

			By("Checking that the Database information in the Synapse Status are correct")
			Expect(s.Status.DatabaseConnectionInfo.ConnectionURL).Should(Equal("unittestdb-primary.unittest-postgres.svc:5432"))
			Expect(s.Status.DatabaseConnectionInfo.DatabaseName).Should(Equal("synapse"))
			Expect(s.Status.DatabaseConnectionInfo.User).Should(Equal("synapse"))
			Expect(s.Status.DatabaseConnectionInfo.Password).Should(Equal(string(base64encode("iOycqrF;EbyqUo7Z2oma}.<L"))))
			//Expect(s.Status.DatabaseConnectionInfo.State).Should(Equal("RUNNING"))
		}

		BeforeEach(func() {
			// Init variables
			r = SynapseReconciler{}
			s = synapsev1alpha1.Synapse{}
			postgresSecret = corev1.Secret{}

			// Init default value the Synapse Status state and for the Secret given as
			// input. These values are intended to be overwritten in the different tests
			// depending on the behavior that is currently being tested.
			synapseDatabaseInfo = synapsev1alpha1.SynapseStatusDatabaseConnectionInfo{}
			postgresSecretData = map[string][]byte{
				"dbname":   []byte("synapse"),
				"host":     []byte("unittestdb-primary.unittest-postgres.svc"),
				"jdbc-uri": []byte("jdbc:postgresql://unittestdb-primary.unittest-postgres.svc:5432/synapse?password=iOycqrF%3BEbyqUo7Z2oma%7D.%3CL&user=synapse"),
				"password": []byte("iOycqrF;EbyqUo7Z2oma}.<L"),
				"port":     []byte("5432"),
				"uri":      []byte("postgresql://synapse:iOycqrF;EbyqUo7Z2oma%7D.%3CL@unittestdb-primary.unittest-postgres.svc:5432/synapse"),
				"user":     []byte("synapse"),
				"verifier": []byte("SCRAM-SHA-256$4096:u1maK6TC+Ti1uiFIf/gp9Q==$/pth+Pn81rgfz6NyJ2kNFNThyLLdgO7iUQcyMlYKIsQ=:5Fvoj/45rznoQByGgK2lcJ9K4PZmYQ5zNOI7kvOoYDg="),
			}
		})

		JustBeforeEach(func() {
			// Configure Synapse Status and PostgreSQL secret
			s.Status.DatabaseConnectionInfo = synapseDatabaseInfo
			postgresSecret.Data = postgresSecretData
		})

		When("when Synapse Status doesn't contain prior database information", func() {
			// Using default values - No BeforeEach node
			It("Should add Database connection information correcty the Synapse Status", check_happy_path)
		})

		When("when homeserver.yaml contain prior database information for a PostgreSQL Instance", func() {
			synapseDatabaseInfo = synapsev1alpha1.SynapseStatusDatabaseConnectionInfo{
				ConnectionURL: "anotherhost-primary.unittest-postgres.svc:2345",
				DatabaseName:  "anotherDB",
				User:          "notsynapse",
				Password:      "PmRJTlF1cn1yPHZKUkUrWmJaRCxkPGE+",
				State:         "RUNNING",
			}
			It(
				"Should overide the existing PostgreSQL Database connection information with the newly created PostgreSQL DB",
				check_happy_path,
			)
		})

		When("when Secret is missing the 'dbname' field", func() {
			BeforeEach(func() {
				delete(postgresSecretData, "dbname")
			})

			It("Should fail to update the ConfigMap data", func() {
				Expect(r.updateSynapseStatusDatabase(&s, postgresSecret)).ShouldNot(Succeed())
			})
		})
	})

	Context("When updating the Synapse ConfigMap Data with PostgreSQL database information", func() {
		var r SynapseReconciler
		var cm corev1.ConfigMap
		var homeserver_in map[interface{}]interface{}
		var homeserver_out map[interface{}]interface{}
		var s synapsev1alpha1.Synapse
		var synapseDatabaseInfo synapsev1alpha1.SynapseStatusDatabaseConnectionInfo

		// Re-usable test for checking different happy paths
		check_happy_path := func() {
			By("Updating the ConfigMap Data")
			Expect(r.updateSynapseConfigMapData(&cm, s)).Should(Succeed())

			By("Parsing the ConfigMap Data and checking Database information are correct")
			configMapData, ok := cm.Data["homeserver.yaml"]
			Expect(ok).Should(BeTrue())

			Expect(yaml.Unmarshal([]byte(configMapData), homeserver_out)).Should(Succeed())

			GinkgoWriter.Println("homeserver_out: ", homeserver_out)
			GinkgoWriter.Println("s: ", s.Status.DatabaseConnectionInfo.ConnectionURL)

			_, ok = homeserver_out["database"]
			Expect(ok).Should(BeTrue())

			GinkgoWriter.Println("homeserver_out[database]: ", homeserver_out["database"])

			//var marschalledHomeserverOutDatabase []byte
			marschalledHomeserverOutDatabase, err := yaml.Marshal(homeserver_out["database"])
			Expect(err).NotTo(HaveOccurred())

			var hs_database HomeserverPgsqlDatabase
			Expect(yaml.Unmarshal(marschalledHomeserverOutDatabase, &hs_database)).To(Succeed())

			// hs_database, ok := homeserver_out["database"].(HomeserverPgsqlDatabase)
			// Expect(ok).Should(BeTrue())

			GinkgoWriter.Println("hs_database: ", hs_database)
			GinkgoWriter.Println("hs_database Name: ", hs_database.Name)

			Expect(hs_database.Name).Should(Equal("psycopg2"))
			Expect(hs_database.Args.User).Should(Equal("synapse"))
			Expect(hs_database.Args.Database).Should(Equal("synapse"))
			Expect(hs_database.Args.Password).Should(Equal("iOycqrF;EbyqUo7Z2oma}.<L+"))
			Expect(hs_database.Args.Host).Should(Equal("unittestdb-primary.unittest-postgres.svc"))
			Expect(hs_database.Args.Port).Should(Equal(int64(5432)))
			Expect(hs_database.Args.CpMin).Should(Equal(int64(5)))
			Expect(hs_database.Args.CpMax).Should(Equal(int64(10)))

			By("Checking that server_name and report_stats values are still present and unchanged")
			_, ok = homeserver_out["server_name"]
			Expect(ok).Should(BeTrue())

			_, ok = homeserver_out["report_stats"]
			Expect(ok).Should(BeTrue())

			server_name, ok := homeserver_out["server_name"].(string)
			Expect(ok).Should(BeTrue())
			Expect(server_name).Should(Equal("example.com"))

			report_stats, ok := homeserver_out["report_stats"].(bool)
			Expect(ok).Should(BeTrue())
			Expect(report_stats).Should(BeTrue())
		}

		BeforeEach(func() {
			// Init variables
			r = SynapseReconciler{}
			cm = corev1.ConfigMap{}
			s = synapsev1alpha1.Synapse{}
			homeserver_out = make(map[interface{}]interface{})

			// Init default value for pre-existing homeserver.yaml, and for Synapse
			// Status given as input. These are intended to be overwritten in the
			// different tests depending on the behavior that is currently being tested.
			homeserver_in = map[interface{}]interface{}{
				"server_name":  "example.com",
				"report_stats": true,
				"database": map[interface{}]interface{}{
					"name": "sqlite3",
					"args": map[interface{}]interface{}{
						"database": "/path/to/homeserver.db",
					},
				},
			}

			synapseDatabaseInfo = synapsev1alpha1.SynapseStatusDatabaseConnectionInfo{
				ConnectionURL: "unittestdb-primary.unittest-postgres.svc:5432",
				DatabaseName:  "synapse",
				User:          "synapse",
				Password:      string(base64encode("iOycqrF;EbyqUo7Z2oma}.<L+")),
				State:         "RUNNING",
			}

		})

		JustBeforeEach(func() {
			// Configure ConfigMap and PostgreSQL secret
			configMapData, err := yaml.Marshal(homeserver_in)
			Expect(err).ShouldNot(HaveOccurred())
			cm.Data = map[string]string{"homeserver.yaml": string(configMapData)}

			s.Status.DatabaseConnectionInfo = synapseDatabaseInfo
		})

		When("when homeserver.yaml doesn't contain prior database information", func() {
			BeforeEach(func() {
				delete(homeserver_in, "database")
			})

			It("Should add Database connection information correcty the ConfigMap", check_happy_path)
		})

		When("when homeserver.yaml contain prior database information for a SQLite DB", func() {
			// Using default values - No BeforeEach node

			It(
				"Should overide the SQLite Database connection information with the newly created PostgreSQL DB",
				check_happy_path,
			)
		})

		When("when homeserver.yaml contain prior database information for a PostgreSQL Instance", func() {
			BeforeEach(func() {
				homeserver_in = map[interface{}]interface{}{
					"server_name":  "example.com",
					"report_stats": true,
					"database": map[interface{}]interface{}{
						"name": "psycopg2",
						"args": map[interface{}]interface{}{
							"user":     "not-synapse",
							"password": "PmRJTlF1cn1yPHZKUkUrWmJaRCxkPGE+",
							"database": "anotherdb",
							"host":     "anotherhost-primary.unittest-postgres.svc",
							"cp_min":   "50",
							"cp_max":   "100",
						},
					},
				}
			})

			It(
				"Should overide the existing PostgreSQL Database connection information with the newly created PostgreSQL DB",
				check_happy_path,
			)
		})

		When("when Synapse Status is missing the database connection information", func() {
			BeforeEach(func() {
				synapseDatabaseInfo = synapsev1alpha1.SynapseStatusDatabaseConnectionInfo{}
			})

			It("Should fail to update the ConfigMap data", func() {
				Expect(r.updateSynapseConfigMapData(&cm, s)).ShouldNot(Succeed())
			})
		})

		When("when Synapse Status database connection information is missing the database name", func() {
			BeforeEach(func() {
				synapseDatabaseInfo.DatabaseName = ""
			})

			It("Should fail to update the ConfigMap data", func() {
				Expect(r.updateSynapseConfigMapData(&cm, s)).ShouldNot(Succeed())
			})
		})

		When("when Synapse Status database connection information is missing the connection URL", func() {
			BeforeEach(func() {
				synapseDatabaseInfo.ConnectionURL = ""
			})

			It("Should fail to update the ConfigMap data", func() {
				Expect(r.updateSynapseConfigMapData(&cm, s)).ShouldNot(Succeed())
			})
		})

		When("when the connection URL is malformed", func() {
			BeforeEach(func() {
				synapseDatabaseInfo.ConnectionURL = "missing.port.url"
			})

			It("Should fail to update the ConfigMap data", func() {
				Expect(r.updateSynapseConfigMapData(&cm, s)).ShouldNot(Succeed())
			})
		})

		When("when Synapse Status database connection information is missing the user", func() {
			BeforeEach(func() {
				synapseDatabaseInfo.User = ""
			})

			It("Should fail to update the ConfigMap data", func() {
				Expect(r.updateSynapseConfigMapData(&cm, s)).ShouldNot(Succeed())
			})
		})

		When("when Synapse Status database connection information is missing the password", func() {
			BeforeEach(func() {
				synapseDatabaseInfo.Password = ""
			})

			It("Should fail to update the ConfigMap data", func() {
				Expect(r.updateSynapseConfigMapData(&cm, s)).ShouldNot(Succeed())
			})
		})
	})
})
