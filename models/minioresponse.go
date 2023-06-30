package models

type Minioresponse struct {
	Records []Records `json:"Records"`
}

type Records struct {
	Bucketinfo Bucketinfo `json:"s3"`
}

type Bucketinfo struct {
	Bucket Bucket `json:"bucket"`
	Object Object `json:"object"`
}

type Bucket struct {
	Name string `json:"name"`
}

type Object struct {
	Key      string   `json:"key"`
	Metadata Metadata `json:"userMetadata"`
}

type Metadata struct {
	Metauuid string `json:"X-Amz-Meta-Uuid"`
}
