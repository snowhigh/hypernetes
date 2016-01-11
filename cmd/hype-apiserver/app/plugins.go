package app

// This file exists to force the desired plugin implementations to be linked.
// This should probably be part of some configuration fed into the build for a
// given binary target.
import (
	// Admission policies
	_ "k8s.io/kubernetes/plugin/pkg/admission/admit"
	_ "k8s.io/kubernetes/plugin/pkg/admission/deny"
	_ "k8s.io/kubernetes/plugin/pkg/admission/exec"
	_ "k8s.io/kubernetes/plugin/pkg/admission/initialresources"
	_ "k8s.io/kubernetes/plugin/pkg/admission/limitranger"
	_ "k8s.io/kubernetes/plugin/pkg/admission/namespace/autoprovision"
	_ "k8s.io/kubernetes/plugin/pkg/admission/namespace/exists"
	_ "k8s.io/kubernetes/plugin/pkg/admission/namespace/lifecycle"
	_ "k8s.io/kubernetes/plugin/pkg/admission/resourcequota"
	_ "k8s.io/kubernetes/plugin/pkg/admission/securitycontext/scdeny"
	_ "k8s.io/kubernetes/plugin/pkg/admission/serviceaccount"
)
