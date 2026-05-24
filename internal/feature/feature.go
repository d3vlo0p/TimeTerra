/*
Copyright 2024.

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

// Package feature provides a lightweight feature-flag registry for TimeTerra.
//
// Flags are enabled at operator startup via the --feature-gates CLI flag or the
// FEATURE_GATES environment variable (comma-separated list of feature names).
// CLI flags take priority over the environment variable.
//
// The special value "ALL" enables every known feature at once:
//
//	--feature-gates=ALL
//
// To add a new feature flag:
//  1. Declare a new const below and append it to All.
//  2. Add a FeatureRegistry *feature.Registry field to the relevant reconciler.
//  3. Gate the experimental code path with r.FeatureRegistry.IsEnabled(feature.MyNewFeature).
//  4. Pass the flag name in --feature-gates when deploying.
package feature

// Feature is the type for a named experimental capability.
// Using a dedicated type (rather than plain string) prevents accidental
// comparisons against unrelated string values.
type Feature string

const (
	// AuroraVerticalScaling enables vertical instance-class scaling for Aurora clusters.
	// When disabled, the "scale" command on AwsRdsAuroraCluster resources is skipped.
	AuroraVerticalScaling Feature = "AuroraVerticalScaling"

	// DocumentDBVerticalScaling enables vertical instance-class scaling for DocumentDB clusters.
	// When disabled, the "scale" command on AwsDocumentDBCluster resources is skipped.
	DocumentDBVerticalScaling Feature = "DocumentDBVerticalScaling"
)

// All is the authoritative list of every known feature flag.
// It is used to validate user-provided flag names and to populate help text.
var All = []Feature{
	AuroraVerticalScaling,
	DocumentDBVerticalScaling,
}

// Registry holds the set of features that are enabled for this operator instance.
// A nil Registry behaves safely: all IsEnabled calls return false.
type Registry struct {
	enabled map[Feature]bool
}

// NewRegistry creates a Registry with exactly the given features enabled.
// Unknown feature names are silently ignored; callers should validate names
// against All before constructing the registry (as done in cmd/main.go).
func NewRegistry(features ...Feature) *Registry {
	r := &Registry{enabled: make(map[Feature]bool, len(features))}
	for _, f := range features {
		r.enabled[f] = true
	}
	return r
}

// IsEnabled reports whether the given feature is active.
// Safe to call on a nil *Registry (returns false).
func (r *Registry) IsEnabled(f Feature) bool {
	if r == nil {
		return false
	}
	return r.enabled[f]
}

// EnabledList returns a slice of all currently enabled features, sorted
// by their order in All. Useful for structured logging at startup.
func (r *Registry) EnabledList() []Feature {
	if r == nil {
		return nil
	}
	out := make([]Feature, 0, len(r.enabled))
	for _, f := range All {
		if r.enabled[f] {
			out = append(out, f)
		}
	}
	return out
}
