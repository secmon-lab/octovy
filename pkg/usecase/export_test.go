package usecase

// Export unexported functions for testing
var (
	DownloadZipFileForTest   = downloadZipFile
	ExtractCodeForTest       = extractCode
	StepDownDirectoryForTest = stepDownDirectory
	ExtractZipFileForTest    = extractZipFile
	UnmarshalFileForTest     = unmarshalFile
	CreateOrUpdateBigQueryTableForTest = createOrUpdateBigQueryTable
)

