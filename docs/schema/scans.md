# BigQuery Scans Table Schema

This document describes the schema of the `scans` table in BigQuery, where Octovy stores Trivy scan results.

## Table Overview

| Property | Value |
|----------|-------|
| Default Dataset | `octovy` |
| Default Table | `scans` |
| Partitioning | By `timestamp` (day) |

Each row represents a single scan execution, containing GitHub metadata and the complete Trivy report.

## Top-Level Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | STRING | Unique scan identifier (UUID) |
| `timestamp` | TIMESTAMP | When the scan was executed |
| `github` | RECORD | GitHub repository and commit metadata |
| `report` | RECORD | Complete Trivy scan report |

## GitHub Metadata (`github`)

Contains information about the scanned repository and commit.

| Field | Type | Description |
|-------|------|-------------|
| `github.repo_id` | INTEGER | GitHub repository ID |
| `github.owner` | STRING | Repository owner (user or organization) |
| `github.repo_name` | STRING | Repository name |
| `github.commit_id` | STRING | Full commit SHA (40 characters) |
| `github.branch` | STRING | Branch name |
| `github.ref` | STRING | Git ref (e.g., `refs/heads/main`) |
| `github.default_branch` | STRING | Default branch of the repository |
| `github.installation_id` | INTEGER | GitHub App installation ID |

### Committer (`github.committer`)

| Field | Type | Description |
|-------|------|-------------|
| `github.committer.id` | INTEGER | GitHub user ID |
| `github.committer.login` | STRING | GitHub username |
| `github.committer.email` | STRING | Email address |

### Pull Request (`github.pull_request`)

Present only when the scan was triggered by a pull request event.

| Field | Type | Description |
|-------|------|-------------|
| `github.pull_request.id` | INTEGER | Pull request ID |
| `github.pull_request.number` | INTEGER | Pull request number |
| `github.pull_request.base_branch` | STRING | Base branch name |
| `github.pull_request.base_commit_id` | STRING | Base commit SHA |
| `github.pull_request.user` | RECORD | PR author (same structure as committer) |

## Trivy Report (`report`)

Contains the complete Trivy scan output.

| Field | Type | Description |
|-------|------|-------------|
| `report.SchemaVersion` | INTEGER | Trivy report schema version |
| `report.ArtifactName` | STRING | Name of the scanned artifact |
| `report.ArtifactType` | STRING | Type of artifact (e.g., `filesystem`) |
| `report.Metadata` | RECORD | Artifact metadata |
| `report.Results` | RECORD (REPEATED) | Array of scan results per target |

### Results (`report.Results[]`)

Each result represents a scan target (e.g., a lock file, container layer).

| Field | Type | Description |
|-------|------|-------------|
| `Target` | STRING | Target identifier (e.g., `go.mod`, `package-lock.json`) |
| `Class` | STRING | Result class (`lang-pkgs`, `os-pkgs`, `config`, `secret`) |
| `Type` | STRING | Package type (e.g., `gomod`, `npm`, `pip`) |
| `Packages` | RECORD (REPEATED) | Detected packages |
| `Vulnerabilities` | RECORD (REPEATED) | Detected vulnerabilities |
| `Misconfigurations` | RECORD (REPEATED) | Detected misconfigurations |
| `Secrets` | RECORD (REPEATED) | Detected secrets |
| `Licenses` | RECORD (REPEATED) | Detected licenses |

### Packages (`report.Results[].Packages[]`)

| Field | Type | Description |
|-------|------|-------------|
| `Name` | STRING | Package name |
| `Version` | STRING | Installed version |
| `Identifier.PURL` | STRING | Package URL (PURL) |
| `Licenses` | STRING (REPEATED) | License identifiers |
| `Indirect` | BOOLEAN | Whether this is an indirect dependency |
| `DependsOn` | STRING (REPEATED) | Direct dependencies |
| `FilePath` | STRING | File path where package is defined |

### Vulnerabilities (`report.Results[].Vulnerabilities[]`)

| Field | Type | Description |
|-------|------|-------------|
| `VulnerabilityID` | STRING | CVE or vendor-specific ID (e.g., `CVE-2021-44228`) |
| `PkgName` | STRING | Affected package name |
| `InstalledVersion` | STRING | Currently installed version |
| `FixedVersion` | STRING | Version that fixes the vulnerability |
| `Severity` | STRING | Severity level (`CRITICAL`, `HIGH`, `MEDIUM`, `LOW`, `UNKNOWN`) |
| `Title` | STRING | Vulnerability title |
| `Description` | STRING | Detailed description |
| `PrimaryURL` | STRING | Primary reference URL |
| `References` | STRING (REPEATED) | Additional reference URLs |
| `CweIDs` | STRING (REPEATED) | CWE identifiers |
| `PublishedDate` | STRING | When the vulnerability was published |
| `LastModifiedDate` | STRING | When the vulnerability was last updated |

## Dynamic Fields

Some fields have dynamic structure based on vulnerability data sources. These fields use map types in the source code, which results in varying field names in BigQuery.

### CVSS Scores (`report.Results[].Vulnerabilities[].CVSS`)

CVSS scores are stored per data source. The field names under `CVSS` vary based on which sources provided scores.

**Common source IDs:**
- `ghsa` - GitHub Security Advisory
- `nvd` - National Vulnerability Database
- `redhat` - Red Hat Security

**Structure per source:**
| Field | Type | Description |
|-------|------|-------------|
| `CVSS.<source>.V2Vector` | STRING | CVSS v2 vector string |
| `CVSS.<source>.V3Vector` | STRING | CVSS v3 vector string |
| `CVSS.<source>.V2Score` | FLOAT | CVSS v2 score (0.0-10.0) |
| `CVSS.<source>.V3Score` | FLOAT | CVSS v3 score (0.0-10.0) |

**Example query for CVSS scores:**
```sql
SELECT
  vuln.VulnerabilityID,
  vuln.CVSS.ghsa.V3Score AS ghsa_score,
  vuln.CVSS.nvd.V3Score AS nvd_score
FROM `your-project.octovy.scans`
CROSS JOIN UNNEST(report.Results) AS result
CROSS JOIN UNNEST(result.Vulnerabilities) AS vuln
WHERE vuln.CVSS.ghsa.V3Score >= 9.0
   OR vuln.CVSS.nvd.V3Score >= 9.0
```

### Vendor Severity (`report.Results[].Vulnerabilities[].VendorSeverity`)

Severity ratings from different sources. Field names are dynamic based on data sources.

**Example:**
```sql
SELECT
  vuln.VulnerabilityID,
  vuln.VendorSeverity.ghsa AS ghsa_severity,
  vuln.VendorSeverity.nvd AS nvd_severity
FROM `your-project.octovy.scans`
CROSS JOIN UNNEST(report.Results) AS result
CROSS JOIN UNNEST(result.Vulnerabilities) AS vuln
```

### Custom Fields

The `Custom` field in vulnerabilities is an extension point and may contain arbitrary JSON. Its structure is not guaranteed and depends on external data sources.

## Query Examples

### Basic Queries

#### Get Latest Scan for Each Repository

```sql
SELECT
  github.owner,
  github.repo_name,
  github.branch,
  timestamp,
  id AS scan_id
FROM `your-project.octovy.scans`
QUALIFY ROW_NUMBER() OVER(PARTITION BY github.owner, github.repo_name ORDER BY timestamp DESC) = 1
ORDER BY timestamp DESC
```

#### Count Scans by Repository

```sql
SELECT
  github.owner,
  github.repo_name,
  COUNT(*) AS scan_count,
  MAX(timestamp) AS last_scan
FROM `your-project.octovy.scans`
GROUP BY github.owner, github.repo_name
ORDER BY scan_count DESC
```

### Vulnerability Queries

#### Find All Critical Vulnerabilities

```sql
SELECT
  github.owner,
  github.repo_name,
  github.commit_id,
  vuln.VulnerabilityID,
  vuln.PkgName,
  vuln.InstalledVersion,
  vuln.FixedVersion,
  vuln.Severity
FROM `your-project.octovy.scans`
CROSS JOIN UNNEST(report.Results) AS result
CROSS JOIN UNNEST(result.Vulnerabilities) AS vuln
WHERE vuln.Severity = 'CRITICAL'
ORDER BY timestamp DESC
```

#### Find Repositories Affected by Specific CVE

```sql
SELECT DISTINCT
  github.owner,
  github.repo_name,
  vuln.PkgName,
  vuln.InstalledVersion,
  vuln.FixedVersion
FROM `your-project.octovy.scans`
CROSS JOIN UNNEST(report.Results) AS result
CROSS JOIN UNNEST(result.Vulnerabilities) AS vuln
WHERE vuln.VulnerabilityID = 'CVE-2021-44228'
```

#### Vulnerability Count by Severity (Last 7 Days)

```sql
SELECT
  vuln.Severity,
  COUNT(*) AS count
FROM `your-project.octovy.scans`
CROSS JOIN UNNEST(report.Results) AS result
CROSS JOIN UNNEST(result.Vulnerabilities) AS vuln
WHERE timestamp > TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 7 DAY)
GROUP BY vuln.Severity
ORDER BY
  CASE vuln.Severity
    WHEN 'CRITICAL' THEN 1
    WHEN 'HIGH' THEN 2
    WHEN 'MEDIUM' THEN 3
    WHEN 'LOW' THEN 4
    ELSE 5
  END
```

#### High CVSS Score Vulnerabilities

```sql
SELECT
  github.owner,
  github.repo_name,
  vuln.VulnerabilityID,
  vuln.PkgName,
  vuln.Severity,
  vuln.CVSS.nvd.V3Score AS cvss_score
FROM `your-project.octovy.scans`
CROSS JOIN UNNEST(report.Results) AS result
CROSS JOIN UNNEST(result.Vulnerabilities) AS vuln
WHERE vuln.CVSS.nvd.V3Score >= 9.0
ORDER BY vuln.CVSS.nvd.V3Score DESC
```

### Package Queries

#### Search for Specific Package (e.g., Log4j)

When a critical vulnerability is announced, find all affected repositories immediately:

```sql
SELECT DISTINCT
  github.owner,
  github.repo_name,
  pkg.Name,
  pkg.Version,
  result.Target
FROM `your-project.octovy.scans`
CROSS JOIN UNNEST(report.Results) AS result
CROSS JOIN UNNEST(result.Packages) AS pkg
WHERE LOWER(pkg.Name) LIKE '%log4j%'
ORDER BY github.owner, github.repo_name
```

#### List All Packages in a Repository

```sql
SELECT DISTINCT
  pkg.Name,
  pkg.Version,
  result.Type AS package_type,
  result.Target
FROM `your-project.octovy.scans` s
CROSS JOIN UNNEST(s.report.Results) AS result
CROSS JOIN UNNEST(result.Packages) AS pkg
WHERE s.github.owner = 'your-org'
  AND s.github.repo_name = 'your-repo'
QUALIFY ROW_NUMBER() OVER (ORDER BY s.timestamp DESC) = 1
ORDER BY result.Target, pkg.Name
```

#### Find Outdated Dependencies with Known Fixes

```sql
SELECT
  github.owner,
  github.repo_name,
  vuln.PkgName,
  vuln.InstalledVersion,
  vuln.FixedVersion,
  vuln.VulnerabilityID
FROM `your-project.octovy.scans`
CROSS JOIN UNNEST(report.Results) AS result
CROSS JOIN UNNEST(result.Vulnerabilities) AS vuln
WHERE vuln.FixedVersion IS NOT NULL
  AND vuln.FixedVersion != ''
  AND timestamp > TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 1 DAY)
ORDER BY github.owner, github.repo_name, vuln.PkgName
```

### Organization-Wide Analysis

#### Vulnerability Summary per Repository

```sql
WITH latest_scans AS (
  SELECT
    github.owner,
    github.repo_name,
    MAX(timestamp) AS latest_timestamp
  FROM `your-project.octovy.scans`
  GROUP BY github.owner, github.repo_name
)
SELECT
  s.github.owner,
  s.github.repo_name,
  COUNTIF(vuln.Severity = 'CRITICAL') AS critical_count,
  COUNTIF(vuln.Severity = 'HIGH') AS high_count,
  COUNTIF(vuln.Severity = 'MEDIUM') AS medium_count,
  COUNTIF(vuln.Severity = 'LOW') AS low_count
FROM `your-project.octovy.scans` s
JOIN latest_scans l
  ON s.github.owner = l.owner
  AND s.github.repo_name = l.repo_name
  AND s.timestamp = l.latest_timestamp
CROSS JOIN UNNEST(s.report.Results) AS result
CROSS JOIN UNNEST(result.Vulnerabilities) AS vuln
GROUP BY s.github.owner, s.github.repo_name
ORDER BY critical_count DESC, high_count DESC
```

#### Most Common Vulnerabilities Across Organization

```sql
SELECT
  vuln.VulnerabilityID,
  vuln.Severity,
  COUNT(DISTINCT CONCAT(github.owner, '/', github.repo_name)) AS affected_repos,
  ANY_VALUE(vuln.Title) AS title
FROM `your-project.octovy.scans`
CROSS JOIN UNNEST(report.Results) AS result
CROSS JOIN UNNEST(result.Vulnerabilities) AS vuln
WHERE timestamp > TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 7 DAY)
GROUP BY vuln.VulnerabilityID, vuln.Severity
ORDER BY affected_repos DESC
LIMIT 20
```

## Best Practices

### Query Performance

1. **Always filter by timestamp** - The table is partitioned by day, so filtering by timestamp reduces scanned data.

2. **Use UNNEST for nested arrays** - Results, Packages, and Vulnerabilities are repeated fields.

3. **Limit scope with WHERE clauses** - Filter by owner/repo early in the query.

### Cost Optimization

1. **Use partitioning** - Queries with `timestamp` filters scan less data.

2. **Select only needed columns** - Avoid `SELECT *` on the full table.

3. **Consider materialized views** - For frequently-run queries, create materialized views.

## See Also

- [BigQuery Setup Guide](../setup/bigquery.md)
- [scan command](../commands/scan.md)
- [insert command](../commands/insert.md)
