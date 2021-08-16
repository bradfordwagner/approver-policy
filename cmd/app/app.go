/*
Copyright 2021 The cert-manager Authors.

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

package app

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	"github.com/cert-manager/policy-approver/cmd/app/options"
	"github.com/cert-manager/policy-approver/pkg/api"
	"github.com/cert-manager/policy-approver/pkg/controllers"
	"github.com/cert-manager/policy-approver/pkg/policy"
)

const (
	helpOutput = "A cert-manager policy approver which bases decisions on CertificateRequestPolicies"
)

func ExecutePolicyApprover() {
	cmd := newCommand(ctrl.SetupSignalHandler())
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func newCommand(ctx context.Context) *cobra.Command {
	opts := new(options.Options)

	cmd := &cobra.Command{
		Use:   "policy-approver",
		Short: helpOutput,
		Long:  helpOutput,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.Complete()
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
				Scheme:                        api.Scheme,
				LeaderElectionNamespace:       opts.LeaderElectionNamespace,
				LeaderElection:                true,
				LeaderElectionID:              "policy.cert-manager.io",
				LeaderElectionReleaseOnCancel: true,
				ReadinessEndpointName:         "/readyz",
				HealthProbeBindAddress:        opts.ReadyzAddress,
				MetricsBindAddress:            opts.MetricsAddress,
				Logger:                        opts.Logr.WithName("controller"),
			})
			if err != nil {
				return fmt.Errorf("unable to start controller manager: %w", err)
			}

			c := controllers.New(
				ctrl.Log, mgr.GetClient(),
				mgr.GetEventRecorderFor("policy-approver"),
				policy.New(mgr.GetClient(), opts.ApproveWhenNoPolicies),
			)
			if err := c.SetupWithManager(mgr); err != nil {
				return fmt.Errorf("unable to create controller CertificateRequestPolicy controller: %w", err)
			}

			if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
				return fmt.Errorf("unable to set up health check: %w", err)
			}
			if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
				return fmt.Errorf("unable to set up ready check: %w", err)
			}

			opts.Logr.WithName("main").Info("starting policy controller")
			return mgr.Start(ctx)
		},
	}

	opts.Prepare(cmd)

	return cmd
}
