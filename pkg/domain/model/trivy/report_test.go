package trivy_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octovy/pkg/domain/model/trivy"
)

func TestReportMarshalUnmarshal(t *testing.T) {
	testCases := []struct {
		name string
		file string
	}{
		{
			name: "Sample Trivy report",
			file: "sample_report.json",
		},
		{
			name: "Real Trivy output",
			file: "real_trivy_output.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Read Trivy output from testdata
			testdataPath := filepath.Join("testdata", tc.file)
			rawData, err := os.ReadFile(testdataPath)
			gt.NoError(t, err)

			// Unmarshal into Report struct
			var report trivy.Report
			err = json.Unmarshal(rawData, &report)
			gt.NoError(t, err)

			// Validate the report
			gt.NoError(t, report.Validate())

			// Re-marshal back to JSON
			remarshaledData, err := json.Marshal(report)
			gt.NoError(t, err)

			// Unmarshal both original and remarshaled to compare
			var original, remarshaledMap map[string]any
			gt.NoError(t, json.Unmarshal(rawData, &original))
			gt.NoError(t, json.Unmarshal(remarshaledData, &remarshaledMap))
			gt.Equal(t, original, remarshaledMap)
		})
	}
}

func TestReportValidation(t *testing.T) {
	t.Run("Valid report passes validation", func(t *testing.T) {
		report := trivy.Report{
			SchemaVersion: 2,
			ArtifactName:  "test-artifact",
			ArtifactType:  "filesystem",
			Results:       trivy.Results{},
		}
		gt.NoError(t, report.Validate())
	})

	t.Run("Invalid schema version fails validation", func(t *testing.T) {
		report := trivy.Report{
			SchemaVersion: 0,
			ArtifactName:  "test-artifact",
			Results:       trivy.Results{},
		}
		err := report.Validate()
		gt.Error(t, err)
	})

	t.Run("Missing artifact name fails validation", func(t *testing.T) {
		report := trivy.Report{
			SchemaVersion: 2,
			ArtifactName:  "",
			Results:       trivy.Results{},
		}
		err := report.Validate()
		gt.Error(t, err)
	})

	t.Run("Nil results fails validation", func(t *testing.T) {
		report := trivy.Report{
			SchemaVersion: 2,
			ArtifactName:  "test-artifact",
			Results:       nil,
		}
		err := report.Validate()
		gt.Error(t, err)
	})

	t.Run("Result with empty target fails validation", func(t *testing.T) {
		report := trivy.Report{
			SchemaVersion: 2,
			ArtifactName:  "test-artifact",
			Results: trivy.Results{
				{
					Target: "",
					Type:   "gomod",
				},
			},
		}
		err := report.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("result target is empty")
		ge := goerr.Unwrap(err)
		gt.V(t, ge.Values()["index"]).Equal(0)
	})

	t.Run("Multiple results with one empty target fails validation", func(t *testing.T) {
		report := trivy.Report{
			SchemaVersion: 2,
			ArtifactName:  "test-artifact",
			Results: trivy.Results{
				{
					Target: "go.mod",
					Type:   "gomod",
				},
				{
					Target: "",
					Type:   "npm",
				},
			},
		}
		err := report.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("result target is empty")
		ge := goerr.Unwrap(err)
		gt.V(t, ge.Values()["index"]).Equal(1)
	})
}

func TestDetectedVulnerabilityID(t *testing.T) {
	t.Run("generates consistent hash from vulnerability data", func(t *testing.T) {
		vuln := &trivy.DetectedVulnerability{
			VulnerabilityID: "CVE-2024-0001",
			PkgName:         "test-package",
			PkgPath:         "/path/to/package",
			PkgID:           "test-package@1.0.0",
		}

		id1 := vuln.ID()
		id2 := vuln.ID()

		// ID should be consistent
		gt.V(t, id1).Equal(id2)
		// ID should be a valid hex string (64 characters for SHA256)
		gt.V(t, len(id1)).Equal(64)
	})

	t.Run("different vulnerabilities have different IDs", func(t *testing.T) {
		vuln1 := &trivy.DetectedVulnerability{
			VulnerabilityID: "CVE-2024-0001",
			PkgName:         "test-package",
			PkgPath:         "/path/to/package",
			PkgID:           "test-package@1.0.0",
		}

		vuln2 := &trivy.DetectedVulnerability{
			VulnerabilityID: "CVE-2024-0002", // Different CVE
			PkgName:         "test-package",
			PkgPath:         "/path/to/package",
			PkgID:           "test-package@1.0.0",
		}

		gt.V(t, vuln1.ID()).NotEqual(vuln2.ID())
	})

	t.Run("same package different paths have different IDs", func(t *testing.T) {
		vuln1 := &trivy.DetectedVulnerability{
			VulnerabilityID: "CVE-2024-0001",
			PkgName:         "test-package",
			PkgPath:         "/path/to/package1",
			PkgID:           "test-package@1.0.0",
		}

		vuln2 := &trivy.DetectedVulnerability{
			VulnerabilityID: "CVE-2024-0001",
			PkgName:         "test-package",
			PkgPath:         "/path/to/package2", // Different path
			PkgID:           "test-package@1.0.0",
		}

		gt.V(t, vuln1.ID()).NotEqual(vuln2.ID())
	})

	t.Run("hash incorporates all required fields", func(t *testing.T) {
		// Test that changing each field changes the hash
		base := &trivy.DetectedVulnerability{
			VulnerabilityID: "CVE-2024-0001",
			PkgName:         "test-package",
			PkgPath:         "/path/to/package",
			PkgID:           "test-package@1.0.0",
		}
		baseID := base.ID()

		// Different VulnerabilityID
		v1 := *base
		v1.VulnerabilityID = "CVE-2024-0002"
		gt.V(t, v1.ID()).NotEqual(baseID)

		// Different PkgName
		v2 := *base
		v2.PkgName = "other-package"
		gt.V(t, v2.ID()).NotEqual(baseID)

		// Different PkgPath
		v3 := *base
		v3.PkgPath = "/other/path"
		gt.V(t, v3.ID()).NotEqual(baseID)

		// Different PkgID
		v4 := *base
		v4.PkgID = "test-package@2.0.0"
		gt.V(t, v4.ID()).NotEqual(baseID)
	})
}
