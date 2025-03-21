package azure

// Config represents the configuration for blobber
type Config struct {
	Accounts            string
	Containers          string
	Download            bool
	Output              string
	SkipSSL             bool
	MaxGoroutines       int
	MaxParallelDownload int
	BaseDomain          string
	Debug               bool
}

// ErrorResponse represents an error response from the Azure blob storage API
type ErrorResponse struct {
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

// BlobProperties represents Azure blob properties
type BlobProperties struct {
	CreationTime     string `xml:"Creation-Time"`
	LastModified     string `xml:"Last-Modified"`
	ContentLength    int64  `xml:"Content-Length"`
	ContentType      string `xml:"Content-Type"`
	ContentMD5       string `xml:"Content-MD5"`
	BlobType         string `xml:"BlobType"`
	AccessTier       string `xml:"AccessTier"`
	LeaseStatus      string `xml:"LeaseStatus"`
	LeaseState       string `xml:"LeaseState"`
	ServerEncrypted  string `xml:"ServerEncrypted"`
}

// Blob represents an Azure blob object
type Blob struct {
	Name       string         `xml:"Name"`
	Properties BlobProperties `xml:"Properties"`
}

// BlobList represents a list of blobs in a container
type BlobList struct {
	Blobs      []Blob  `xml:"Blobs>Blob"`
	NextMarker string  `xml:"NextMarker"`
}

// EnumerationResults represents blob list returned from the Azure API
type EnumerationResults struct {
	ServiceEndpoint string   `xml:"ServiceEndpoint,attr"`
	ContainerName   string   `xml:"ContainerName,attr"`
	BlobList        BlobList
}

// AccessResult represents an access result for a container
type AccessResult struct {
	Account   string
	Container string
	IsPublic  bool
	ErrorCode string
	URL       string
	Blobs     []string
} 