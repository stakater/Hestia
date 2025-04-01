package utils

import (
	"fmt"
	"github.com/gkampitakis/go-snaps/match"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/dsl/core"
	"github.com/stakater/hestia-operator/internal/constants"
	"k8s.io/apimachinery/pkg/runtime"
	"strings"
)

var excludeFields = []string{
	"$.metadata.uid",
	"$.metadata.resourceVersion",
	"$.metadata.creationTimestamp",
	"$.metadata.annotations",
	"$.metadata.managedFields",
	"$.metadata.ownerReferences",
	"$.metadata.name",
	"$.spec.selector",
	"$.spec.template.metadata",
	"$.status.startTime",
	"$.status.completionTime",
	"$.status.conditions[*].lastTransitionTime",
	"$.status.conditions[*].lastProbeTime",
	"$.status.lastSuccessfulRunTime",
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
	snaps.WithConfig(
		snaps.Dir(fmt.Sprintf("__snapshots__/%s", currentSpec.FullText())),
		snaps.Filename(name),
		snaps.Ext(".yaml"),
	).MatchYAML(
		core.GinkgoT(),
		resource,
		match.Any(excludeFields...).ErrOnMissingPath(false),
		match.Custom("$.metadata.labels", func(val any) (any, error) {
			replaceMapValue(val)
			return val, nil
		}),
	)
}
