package performanceprofile

import (
	"testing"

	performancev2 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/performanceprofile/v2"
	"github.com/stretchr/testify/assert"
)

func cpuSet(value string) *performancev2.CPUSet {
	set := performancev2.CPUSet(value)
	return &set
}

func TestValidatePerformanceProfileRejectsMissingCPUFields(t *testing.T) {
	tests := []struct {
		name    string
		profile *performancev2.PerformanceProfile
	}{
		{
			name:    "nil profile",
			profile: nil,
		},
		{
			name:    "nil CPU",
			profile: &performancev2.PerformanceProfile{},
		},
		{
			name: "nil isolated CPU set",
			profile: &performancev2.PerformanceProfile{
				Spec: performancev2.PerformanceProfileSpec{
					CPU: &performancev2.CPU{},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			valid, err := ValidatePerformanceProfile(test.profile)

			assert.False(t, valid)
			assert.NoError(t, err)
		})
	}
}

func TestValidatePerformanceProfile(t *testing.T) {
	tests := []struct {
		name          string
		profile       *performancev2.PerformanceProfile
		wantValid     bool
		wantErr       bool
		wantHugePages string
	}{
		{
			name: "invalid CPU set",
			profile: &performancev2.PerformanceProfile{
				Spec: performancev2.PerformanceProfileSpec{
					CPU: &performancev2.CPU{Isolated: cpuSet("invalid")},
				},
			},
			wantErr: true,
		},
		{
			name: "too few isolated CPUs",
			profile: &performancev2.PerformanceProfile{
				Spec: performancev2.PerformanceProfileSpec{
					CPU: &performancev2.CPU{Isolated: cpuSet("0-4")},
				},
			},
		},
		{
			name: "missing hugepages",
			profile: &performancev2.PerformanceProfile{
				Spec: performancev2.PerformanceProfileSpec{
					CPU: &performancev2.CPU{Isolated: cpuSet("0-5")},
				},
			},
		},
		{
			name: "valid x86 profile",
			profile: &performancev2.PerformanceProfile{
				Spec: performancev2.PerformanceProfileSpec{
					CPU: &performancev2.CPU{Isolated: cpuSet("0-5")},
					HugePages: &performancev2.HugePages{Pages: []performancev2.HugePage{{
						Size:  performancev2.HugePageSize(X86PerformanceProfileHugepageSize),
						Count: 10,
					}}},
				},
			},
			wantValid:     true,
			wantHugePages: X86PerformanceProfileHugepageSize,
		},
		{
			name: "valid arm profile",
			profile: &performancev2.PerformanceProfile{
				Spec: performancev2.PerformanceProfileSpec{
					CPU: &performancev2.CPU{Isolated: cpuSet("0-5")},
					HugePages: &performancev2.HugePages{Pages: []performancev2.HugePage{{
						Size:  performancev2.HugePageSize(Arm64KPerformanceProfileHugepageSize),
						Count: 4,
					}}},
				},
			},
			wantValid:     true,
			wantHugePages: Arm64KPerformanceProfileHugepageSize,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			HugePageSize = ""

			valid, err := ValidatePerformanceProfile(test.profile)

			assert.Equal(t, test.wantValid, valid)
			if test.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, test.wantHugePages, HugePageSize)
		})
	}
}
