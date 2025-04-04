package utils

import (
	"fmt"
	"github.com/gkampitakis/go-snaps/match"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/dsl/core"
	"github.com/stakater/hestia-operator/internal/constants"
	"k8s.io/apimachinery/pkg/runtime"
	"path/filepath"
	"strings"
)

var excludeFields = []string{
	"$.metadata.uid",
	"$.metadata.resourceVersion",
	"$.metadata.creationTimestamp",
	"$.metadata.annotations",
	"$.metadata.managedFields",
	"$.metadata.ownerReferences",
	"$.spec.selector",
	"$.spec.template.metadata",
	"$.status.startTime",
	"$.status.completionTime",
	"$.status.conditions[*].lastTransitionTime",
	"$.status.conditions[*].lastProbeTime",
	"$.status.lastSuccessfulRunTime",
	"$.status.uncountedTerminatedPods",
	"$.status.lastScheduleTime",
	"$.status.lastSuccessfulTime",
}

var excludeJobFields = append(excludeFields, []string{
	"$.metadata.name",
}...)

var excludeFieldMap = map[string][]string{
	"job": excludeJobFields,
}

func replaceMapValue(val any) {
	labels, ok := val.(map[string]interface{})
	if !ok {
		return
	}

	if _, exists := labels[constants.VersionLabel]; exists {
		labels[constants.VersionLabel] = "<Any value>"
	}
}

func MatchYAMLResource(resource runtime.Object, snapshotName ...string) {
	kind := strings.ToLower(GetKind(resource))
	currentSpec := ginkgo.CurrentSpecReport()
	name := strings.Join(snapshotName, "_")
	if name == "" {
		name = currentSpec.LeafNodeText
	}

	name = fmt.Sprintf("[%s] %s", kind, name)
	exclude, ok := excludeFieldMap[kind]
	if !ok {
		exclude = excludeFields
	}

	snaps.WithConfig(
		snaps.Dir(fmt.Sprintf("__snapshots__/%s/%s", filepath.Base(currentSpec.FileName()), currentSpec.LeafNodeText)),
		snaps.Filename(name),
		snaps.Ext(".yaml"),
	).MatchYAML(
		core.GinkgoT(),
		resource,
		match.Any(exclude...).ErrOnMissingPath(false),
		match.Custom("$.metadata.labels", func(val any) (any, error) {
			replaceMapValue(val)
			return val, nil
		}),
		match.Custom("$.spec.jobTemplate.metadata.labels", func(val any) (any, error) {
			replaceMapValue(val)
			return val, nil
		}).ErrOnMissingPath(false),
	)
}
